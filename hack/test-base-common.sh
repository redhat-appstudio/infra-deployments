#!/usr/bin/env bash
#
# test-base-common.sh - Base common-ground check for -rd ring-deployment migrations.
#
# Verifies that components/<comp>-rd/base/ captures the configuration common to all
# environments (development / staging / production), and flags common config that
# was left out of the base. Local reviewer aid for KFLUXINFRA-4497; does not touch
# clusters or run chainsaw.
#
# Usage:
#   hack/test-base-common.sh <component-rd>
#
# The unit of comparison is a whole rendered document (one Kubernetes object).
# Comparison is never done below the document level. A document is "common" when it
# is byte-identical across every environment that exists for the component.
#
# Exit 0 if base captures all common ground, 1 otherwise (or on setup error).

set -euo pipefail

MAX_DIFF_LINES="${MAX_DIFF_LINES:-80}"

usage() { echo "Usage: $0 <component-rd>" >&2; exit 2; }
[ $# -eq 1 ] || usage

COMPONENT_RD="$1"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

COMPONENT="${COMPONENT_RD%-rd}"
if [ "$COMPONENT" = "$COMPONENT_RD" ]; then
  echo "ERROR: component '$COMPONENT_RD' does not end in '-rd'." >&2
  exit 1
fi

RD_BASE="components/${COMPONENT_RD}/base"
ORIG_DIR="components/${COMPONENT}"
[ -d "$RD_BASE" ] || { echo "ERROR: $RD_BASE not found." >&2; exit 1; }
[ -d "$ORIG_DIR" ] || { echo "ERROR: $ORIG_DIR not found." >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PROBLEMS=0

build() { kustomize build --enable-helm "$1"; }

# env_top <env> -> deployment/base overlay dir for an env, or empty if absent.
env_top() {
  local env="$1"
  if [ -f "$ORIG_DIR/$env/base/kustomization.yaml" ]; then
    echo "$ORIG_DIR/$env/base"
  elif [ -f "$ORIG_DIR/$env/kustomization.yaml" ]; then
    echo "$ORIG_DIR/$env"
  else
    echo ""
  fi
}

# split_build <kustomize-path> <tag>
# Builds the path, splits the output into per-document files under $WORK/<tag>/,
# and writes an index "<hash>\t<kind/name>" (one line per document) to
# $WORK/<tag>.idx. Also records hash->representative-doc under $WORK/docs/<hash>.
# Returns nonzero (and prints a marker) if the build fails.
split_build() {
  local path="$1" tag="$2"
  local dir="$WORK/$tag"
  # Clear any docs from a previous call reusing this tag (e.g. 'basesub' in the
  # Pass 2 loop); otherwise stale doc_*.yaml would be re-indexed.
  rm -rf "$dir"
  mkdir -p "$dir" "$WORK/docs"
  : > "$WORK/$tag.idx"

  local stream="$WORK/$tag.stream"
  if ! build "$path" > "$stream" 2>"$WORK/$tag.err"; then
    return 1
  fi

  # Split the multi-document YAML stream on lines that are exactly '---'.
  awk -v out="$dir" '
    BEGIN { n = 0; f = sprintf("%s/doc_%05d.yaml", out, n) }
    /^---[[:space:]]*$/ { n++; f = sprintf("%s/doc_%05d.yaml", out, n); next }
    { print > f }
  ' "$stream"

  local doc h kind name label
  for doc in "$dir"/doc_*.yaml; do
    [ -f "$doc" ] || continue
    # Skip documents that are empty / whitespace only.
    if ! grep -q '[^[:space:]]' "$doc"; then continue; fi
    h="$(sha256sum "$doc" | cut -d' ' -f1)"
    kind="$(awk -F': *' '/^kind:/ {print $2; exit}' "$doc")"
    name="$(awk -F': *' '/^  name:/ {print $2; exit}' "$doc")"
    label="${kind:-?}/${name:-?}"
    printf '%s\t%s\n' "$h" "$label" >> "$WORK/$tag.idx"
    [ -f "$WORK/docs/$h" ] || cp "$doc" "$WORK/docs/$h"
  done
  return 0
}

# hashes <tag> -> sorted unique hashes for a built tag.
hashes() { cut -f1 "$WORK/$1.idx" | sort -u; }

# label_for <hash> -> a human label for a document hash.
label_for() { grep -m1 -F "$1" "$WORK"/*.idx | head -1 | cut -f2; }

# show_doc <hash> - print the document, indented and truncated.
show_doc() {
  local h="$1" n
  n="$(wc -l < "$WORK/docs/$h")"
  if [ "$n" -gt "$MAX_DIFF_LINES" ]; then
    head -n "$MAX_DIFF_LINES" "$WORK/docs/$h" | sed 's/^/        /'
    echo "        ... $((n - MAX_DIFF_LINES)) more lines - too many to show"
  else
    sed 's/^/        /' "$WORK/docs/$h"
  fi
}

# intersect_all <out-file> <tag1> <tag2> ... - hashes present in every tag.
intersect_all() {
  local out="$1"; shift
  local n=$#
  # Count how many tags each hash appears in; keep those in all n.
  { for t in "$@"; do hashes "$t"; done; } | sort | uniq -c \
    | awk -v n="$n" '$1 == n { print $2 }' | sort -u > "$out"
}

echo "Component: $COMPONENT_RD (original: $COMPONENT)"
echo "Base:      $RD_BASE"
echo

# Resolve env-tops.
ENVS=()
ENV_TAGS=()
for e in development staging production; do
  top="$(env_top "$e")"
  if [ -n "$top" ]; then
    tag="env_$e"
    if split_build "$top" "$tag"; then
      ENVS+=("$top")
      ENV_TAGS+=("$tag")
      echo "env: $e -> $top"
    else
      echo "FAIL: kustomize build failed for $top"
      sed 's/^/      /' "$WORK/$tag.err" >&2 || true
      PROBLEMS=$((PROBLEMS + 1))
    fi
  fi
done
echo

if [ "${#ENV_TAGS[@]}" -lt 2 ]; then
  echo "ERROR: need at least 2 environments to compute common ground (found ${#ENV_TAGS[@]})." >&2
  exit 1
fi

# Build the rd base once.
if ! split_build "$RD_BASE" "base"; then
  echo "FAIL: kustomize build failed for $RD_BASE" >&2
  sed 's/^/      /' "$WORK/base.err" >&2 || true
  exit 1
fi

# Optional advisory cross-check: original global base, if present. This is
# informational only and never a failure - a valid migration may deliberately
# reorganize the original base (e.g. version-pinned upstream resources kept in
# per-ring snapshots rather than the shared base), so byte-identity is not
# expected. The authoritative signal is the Pass 1 document intersection below.
if [ -f "$ORIG_DIR/base/kustomization.yaml" ]; then
  echo "Note: original global base exists ($ORIG_DIR/base); comparing rd base against it (advisory)."
  if split_build "$ORIG_DIR/base" "origbase"; then
    d="$(diff -u --label "$ORIG_DIR/base" --label "$RD_BASE" \
          "$WORK/origbase.stream" "$WORK/base.stream" || true)"
    if [ -z "$d" ]; then
      echo "  rd base == original base (identical render)."
    else
      echo "  rd base DIFFERS from original base (advisory, not a failure):"
      n="$(printf '%s\n' "$d" | wc -l)"
      if [ "$n" -gt "$MAX_DIFF_LINES" ]; then
        # awk (not head) consumes all input, so the upstream printf never gets
        # SIGPIPE under 'set -o pipefail'.
        printf '%s\n' "$d" | awk -v m="$MAX_DIFF_LINES" 'NR <= m' | sed 's/^/    /'
        echo "    ... $((n - MAX_DIFF_LINES)) more lines - too many changes to show"
      else
        printf '%s\n' "$d" | sed 's/^/    /'
      fi
    fi
  fi
  echo
fi

# ---- Pass 1: whole-build intersection (authoritative) ----
echo "=== Pass 1: whole-build common ground vs base ==="
intersect_all "$WORK/common.hashes" "${ENV_TAGS[@]}"
hashes base > "$WORK/base.hashes"

comm -23 "$WORK/common.hashes" "$WORK/base.hashes" > "$WORK/missing.hashes"

n_common="$(wc -l < "$WORK/common.hashes")"
n_missing="$(wc -l < "$WORK/missing.hashes")"
echo "common documents across all envs: $n_common"
echo "  captured in base:              $((n_common - n_missing))"
echo "  MISSING from base:            $n_missing"
echo

if [ "$n_missing" -gt 0 ]; then
  echo "Common documents missing from base:"
  while read -r h; do
    [ -n "$h" ] || continue
    echo "  - $(label_for "$h")"
  done < "$WORK/missing.hashes"
  echo "  (review aid: some of these may be intentional - e.g. version-pinned"
  echo "   upstream resources kept in per-ring snapshots rather than the base.)"
  PROBLEMS=$((PROBLEMS + 1))
  echo
fi

# ---- Pass 2: per-subfolder attribution ----
echo "=== Pass 2: sub-folders with common ground absent from base ==="

# Subfolder names present at EVERY env-top.
: > "$WORK/subdirs.all"
for top in "${ENVS[@]}"; do
  for d in "$top"/*/; do
    if [ -d "$d" ]; then basename "$d"; fi
  done | sort -u
done > "$WORK/subdirs.all"
n_envs="${#ENVS[@]}"
mapfile -t COMMON_SUBS < <(sort "$WORK/subdirs.all" | uniq -c \
  | awk -v n="$n_envs" '$1 == n { print $2 }')

flagged=0
for sub in "${COMMON_SUBS[@]}"; do
  # Build the subfolder in each env; intersect its documents.
  sub_tags=()
  ok=1
  i=0
  for top in "${ENVS[@]}"; do
    tag="sub_${sub}_${i}"
    if split_build "$top/$sub" "$tag"; then
      sub_tags+=("$tag")
    else
      ok=0
    fi
    i=$((i + 1))
  done
  [ "$ok" -eq 1 ] || { echo "  WARN: build failed for sub-folder '$sub' in some env; skipping"; continue; }

  intersect_all "$WORK/sub_common.hashes" "${sub_tags[@]}"

  # Base's copy of this subfolder (may be absent).
  if [ -d "$RD_BASE/$sub" ] && split_build "$RD_BASE/$sub" "basesub"; then
    hashes basesub > "$WORK/basesub.hashes"
  else
    : > "$WORK/basesub.hashes"
  fi

  comm -23 "$WORK/sub_common.hashes" "$WORK/basesub.hashes" > "$WORK/sub_missing.hashes"
  n_sub_missing="$(wc -l < "$WORK/sub_missing.hashes")"

  if [ "$n_sub_missing" -gt 0 ]; then
    flagged=1
    if [ -d "$RD_BASE/$sub" ]; then
      echo "  [$sub] $n_sub_missing common document(s) not in base/$sub:"
    else
      echo "  [$sub] present in all envs but ABSENT from base ($n_sub_missing common document(s)):"
    fi
    while read -r h; do
      [ -n "$h" ] || continue
      echo "      - $(label_for "$h")"
    done < "$WORK/sub_missing.hashes"
    PROBLEMS=$((PROBLEMS + 1))
  fi
done

if [ "$flagged" -eq 0 ]; then
  echo "  none - base captures the common ground of every shared sub-folder."
fi
echo

echo "Summary: $PROBLEMS problem group(s) found."
[ "$PROBLEMS" -eq 0 ]
