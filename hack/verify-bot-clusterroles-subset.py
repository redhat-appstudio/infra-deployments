#!/usr/bin/env python3
"""Verify konflux-*-bot-actions ClusterRoles are a subset of konflux-admin-user-actions.

Namespace admins bind the bot ClusterRoles to their konflux-bot-* ServiceAccounts.
The apiserver's privilege-escalation check refuses a RoleBinding unless the creator
already holds every permission in the referenced role, so any permission that lands
in a bot role but not in konflux-admin-user-actions silently breaks that workflow:

  rolebindings.rbac.authorization.k8s.io "..." is forbidden: user "..." is
  attempting to grant RBAC permissions not currently held: {...}

Granting admins the bind verb would waive that check, but bind is not subject-scoped
-- the restrict-bindings-serviceaccounts VAP only constrains konflux-bot-* subjects,
so an admin could bind a bot role to themselves and pick up anything the admin role
lacks. Keeping the bot roles a subset avoids needing bind at all.

Two sources ship these roles. components/konflux-rbac/<env>/<cluster>/ serves the
clusters Argo CD targets directly; operator-owned clusters are excluded from that
ApplicationSet (see argo-cd-apps/overlays/*/exclude-operator-owned-clusters.yaml)
and get theirs from components/konflux-operator/rings/*/base/cr/konflux-rbac/
instead. Both must satisfy the invariant, so both are checked.

Usage:
    hack/verify-bot-clusterroles-subset.py <dir> [<dir> ...]
    hack/verify-bot-clusterroles-subset.py --admin-from <dir> <dir> [<dir> ...]

--admin-from names a directory to source konflux-admin-user-actions from, for
targets that ship bot roles without an admin role. The konflux-operator CR
directories are such targets: on those clusters the admin role is supplied by the
konflux-operator image rather than this repo, so the in-repo role is used as a
stand-in. That is a proxy, not a guarantee -- see the note in the workflow.

Exits non-zero and lists the offending permissions if any bot role exceeds the
admin role. Set OUTPUT=GITHUB to emit GitHub Actions error annotations.
"""

import argparse
import os
import subprocess
import sys

import yaml

ADMIN_ROLE = "konflux-admin-user-actions"
BOT_ROLE_SUFFIX = "-bot-actions"
# konflux-tester-internalbot-actions and friends are not bindable to konflux-bot-*
# ServiceAccounts (they do not match the VAP's ^konflux-.+-bot-actions$ pattern),
# so admins never need to bind them and they are exempt from this check.
EXEMPT_SUBSTRING = "internalbot"


def render(path):
    """Return {clusterrole name: rules} for a kustomize overlay or component dir."""
    result = subprocess.run(
        ["kustomize", "build", "--enable-helm", path],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"kustomize build failed for {path}:\n{result.stderr}")
    roles = {}
    for doc in yaml.safe_load_all(result.stdout):
        if doc and doc.get("kind") == "ClusterRole":
            roles[doc["metadata"]["name"]] = doc.get("rules") or []
    return roles


def expand(rules):
    """Flatten rules into (apiGroup, resource, verb) triples.

    Rules narrowed by resourceNames are skipped: they grant strictly less than the
    same rule unscoped, and treating them as full grants would produce false
    positives. This makes the check conservative in the safe direction -- it can
    miss a narrow grant, never invent one.
    """
    perms = set()
    for rule in rules:
        if rule.get("resourceNames"):
            continue
        for group in rule.get("apiGroups", []):
            for resource in rule.get("resources", []):
                for verb in rule.get("verbs", []):
                    perms.add((group, resource, verb))
    return perms


def covered_by(admin_perms, perm):
    group, resource, verb = perm
    return any(
        (ag in (group, "*")) and (ares in (resource, "*")) and (averb in (verb, "*"))
        for ag, ares, averb in admin_perms
    )


def check(target, admin_perms, admin_source, github):
    """Report bot roles in target exceeding admin_perms. Returns failure count."""
    roles = render(target)
    failures = 0
    for name in sorted(roles):
        if not name.endswith(BOT_ROLE_SUFFIX) or EXEMPT_SUBSTRING in name:
            continue
        excess = sorted(p for p in expand(roles[name]) if not covered_by(admin_perms, p))
        if not excess:
            continue
        failures += 1
        detail = ", ".join(f"{v} {r}.{g or 'core'}" for g, r, v in excess)
        via = "" if admin_source == target else f" (compared against {admin_source})"
        msg = (
            f"{target}: {name} grants {len(excess)} permission(s) not held by "
            f"{ADMIN_ROLE}{via}: {detail}. Admins will not be able to bind this "
            f"role to konflux-bot-* ServiceAccounts. Add the permission to "
            f"{ADMIN_ROLE} or remove it from the bot role."
        )
        if github:
            print(f"::error title=Bot ClusterRole exceeds admin role::{msg}")
        else:
            print(f"ERROR: {msg}", file=sys.stderr)
    return failures


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("targets", nargs="+", help="kustomize overlay or component dirs")
    parser.add_argument("--admin-from", metavar="DIR",
                        help=f"source {ADMIN_ROLE} from DIR instead of from each target")
    args = parser.parse_args()
    github = os.environ.get("OUTPUT") == "GITHUB"

    shared_perms = None
    if args.admin_from:
        roles = render(args.admin_from)
        if ADMIN_ROLE not in roles:
            print(f"ERROR: {args.admin_from} does not define {ADMIN_ROLE}", file=sys.stderr)
            return 2
        shared_perms = expand(roles[ADMIN_ROLE])

    failures = 0
    checked = 0
    for target in args.targets:
        if shared_perms is not None:
            admin_perms, admin_source = shared_perms, args.admin_from
        else:
            roles = render(target)
            if ADMIN_ROLE not in roles:
                print(f"ERROR: {target} defines no {ADMIN_ROLE}; pass --admin-from",
                      file=sys.stderr)
                return 2
            admin_perms, admin_source = expand(roles[ADMIN_ROLE]), target
        failures += check(target, admin_perms, admin_source, github)
        checked += 1

    if failures:
        return 1
    print(f"OK: all bot ClusterRoles are a subset of {ADMIN_ROLE} "
          f"across {checked} target(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
