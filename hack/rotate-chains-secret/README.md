# Rotate Tekton Chains keys by ring

Rotate a separate Cosign private key, public key, and password for each cluster
in **one pipeline-service ring per run**. The playbook writes new Vault KV v2
versions and aligns legacy ExternalSecret paths with the new-cluster convention:

```text
stonesoup/<environment>/platform/ansible/generated/<cluster>/chains-signing-secret
```

Key generation, upload, verification, and cleanup reuse the
[new-cluster Chains tasks](../new-cluster/tasks/vault/chains-key.yaml).
Bootstrap retains its create-only check; this playbook supplies the ring's
preflighted versions and recovery journal to the same shared tasks.

Ring membership comes from `components/pipeline-service/rings/<ring>/`.
The playbook renders each selected cluster's overlay to find its current Vault
source; it does not infer legacy paths from cluster names. Ring 1 is staging;
rings 2–4 are production. Ring 0 uses an in-cluster key-generation Job and is
not supported by this Vault playbook.

## Prerequisites

- Ansible, Kustomize, Cosign, OpenSSL, and the Vault CLI on the local machine.
- VPN access and Vault permissions for metadata read, secret
  create/update, and secret read for the selected paths in the
  `stonesoup` KV v2 engine. Only the public key is requested during readback.
- For rollback, retention of at least two Vault versions. Preflight rejects an
  explicit per-key retention limit of one; inherited engine retention settings
  are not queried or validated.
- A clean checkout of the relevant pipeline-service overlays from `main`.
  Existing changes to other files are not part of the rotation.
- A plan for distributing the new public keys to signature verifiers and
  retaining the old public keys wherever historical signatures must remain
  verifiable. This playbook replaces the current `cosign.pub`; it does not
  manage verifier trust stores.

The playbook checks the existing Vault session before preflighting any cluster.
If the session is missing or invalid, it reuses the new-cluster OIDC login tasks
to open your browser for SSO. Complete the browser login to continue. This also
applies to `--check`; authentication does not rotate any keys.

A valid session from `VAULT_TOKEN` or the Vault CLI's token helper is reused
without opening a browser. After SSO, the playbook uses the refreshed token
helper and ignores an expired `VAULT_TOKEN` for the rest of that run. Login
responses and tokens are hidden from Ansible logs. A failed login stops the run
before key generation or Vault secret writes.

To authenticate manually, use the same Vault address as the playbook:

```bash
export VAULT_ADDR=https://vault.devshift.net
vault login -method=oidc -no-print
```

Set `VAULT_ADDR` or `-e vault_addr=...` to override the default server.

## Preview one ring

Run from the repository root:

```bash
ansible-playbook hack/rotate-chains-secret/playbook.yaml -e ring=ring-1 --check
```

Check mode renders the selected overlays, validates that both ExternalSecrets
use the same unpinned source, and reads Vault metadata for every selected
cluster. It prints the source path/version and canonical destination. It does
not generate keys, write to Vault, create a recovery journal, or edit manifests.
Missing, deleted, or destroyed current versions abort the run. Metadata access,
retention, or configuration errors in any selected cluster abort preflight
before key generation begins. Write permissions are exercised during upload;
a write failure stops the ring at that cluster.

## Rotate and align the selected ring

```bash
ansible-playbook hack/rotate-chains-secret/playbook.yaml -e ring=ring-1
```

The playbook processes clusters sequentially, stopping on the first failure:

1. Generate a fresh password and encrypted Cosign key pair in a private
   temporary directory.
2. Record the source path/version and expected destination version in a local
   JSON Lines recovery journal.
3. Upload `cosign.key`, `cosign.pub`, and `cosign.password` with a Vault CAS
   check against the destination version inspected during preflight, or zero
   when the destination is absent. An existing canonical destination is rotated
   before updating the local ExternalSecret paths.
4. Record the successful write and verify the new public key by reading back
   that exact version.
5. For a migrated path, update **both local** signing/public-key patch files
   under the selected ring. The old Vault path is left intact.
6. Remove the temporary key material, including on handled failures.

For **already aligned clusters**, the Vault write changes the current version
immediately. External Secrets will reconcile it on its refresh interval
(currently five minutes). Signing and public-key Secrets reconcile separately;
validate both before treating the cluster as ready.

For **migrated clusters**, the live ExternalSecrets keep using the old path
until the manifest PR is merged and synced. Review `git diff`, then submit the
path changes as a PR for this ring only. Follow the repository's
[PR workflow](../../skills/pr-workflow.md); production PRs need a Risk
Assessment and Validation section. The playbook does not commit, push, or
open a PR.

The journal path is printed before rotation. To select a persistent location:

```bash
ansible-playbook hack/rotate-chains-secret/playbook.yaml \
  -e ring=ring-1 -e rotation_log=/tmp/chains-ring-1.jsonl
```

Journal entries contain paths, versions, and a public-key SHA-256 fingerprint,
never private keys or passwords. The fingerprint hashes the PEM text with
leading/trailing whitespace removed.
Keep the journal until the rollout and rollback window are complete. A journal
entry marked `attempting` without a later `written` entry means the result of
the write must be checked in Vault before retrying. Compare the public-key
fingerprint to distinguish this run's key from a concurrent writer's key.

To run a canary or resume only the remaining clusters, supply a JSON list:

```bash
ansible-playbook hack/rotate-chains-secret/playbook.yaml \
  -e '{"ring":"ring-1","clusters":["kflux-stg-p02"]}' --check
```

Remove `--check` to execute that selection. Names outside the selected ring,
duplicate names, shared source paths, and destinations used by another selected
cluster are rejected. Each successful run
generates fresh keys: rerunning a completed aligned cluster rotates it again.
An existing canonical destination is rotated using its inspected version even
when the local overlay still uses a legacy path. If the upload succeeded but
the local patch update did not, verify the journal fingerprint,
complete both path patches for that cluster, and exclude it from the retry's
cluster list.

## Validate before advancing

After the ring's manifest changes have synced, check each selected cluster:

```bash
oc --context=<cluster-context> -n openshift-pipelines get externalsecret \
  tekton-chains-signing-secret tekton-chains-public-key
```

Confirm both are Ready, use the canonical source path, and have reconciled
after the upload. Compare the public key in Vault with the cluster's
`public-key` Secret, and verify a newly completed build's signature with that
public key. Check Chains controller logs for signing errors. Keep the old
public keys available to verifiers of older artifacts.

Finish and validate ring 1 before explicitly invoking ring 2, then ring 3,
then ring 4. No invocation advances to another ring automatically.

## Recovery

Earlier successful clusters are not automatically rolled back when a later
cluster fails. Inspect the journal, Vault metadata, and local `git diff` first.
An interrupted process or a machine crash can leave its temporary directory
behind; remove the `chains-key-*` directory printed in that run once recovery
is understood. Normal Ansible failures run the cleanup block.

For an **in-place rotation**, use the journal's `previous_version` to create a
new current Vault version containing the previous key material, for example:

```bash
vault kv rollback -mount=stonesoup -version=4 \
  production/platform/ansible/generated/<cluster>/chains-signing-secret
```

The previous version must still exist and be readable under Vault's retention
policy. Wait for both Secrets to reconcile and validate signing again. A CAS
failure indicates a concurrent update; inspect that version before choosing
what to restore.

For a **path migration**, the journal records `source_path` and
`source_version`. Before the manifest PR is merged, the old key remains active
and there is no live path change to undo. After merge, revert the selected
ring's path changes through a PR to restore the untouched legacy source.
Do not remove the old paths until the rollback window and historical-key
requirements have been addressed.

## Offline verification

With Ansible and PyYAML installed:

```bash
python3 -m unittest discover -s hack/rotate-chains-secret -v
yamllint --no-warnings hack/rotate-chains-secret
ansible-playbook hack/rotate-chains-secret/playbook.yaml --syntax-check
```

The tests execute the real playbook with fake Kustomize, Vault, Cosign, and
OpenSSL executables in temporary directories. They do not contact Vault or
use real credentials.
