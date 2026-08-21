# Application-layer new-cluster automation

This is step 2 of 3 in automating the rollout of a new cluster.

This automation creates yaml files in the infra-deployments repo locally.

## Prerequisite

1. The cluster-layer terraform automation has been run, and the new cluster is up. This entails that the AWS, IBM, and database secrets have been provisioned and injected into vault in the correct paths. This automation will verify that before proceeding.

2. You need the following installed on your system:

* ansible CLI
* vault CLI
* python3-hvac
* python-requests

3. You have variables that correctly describe the new cluster,

* `longname` - Example: `kflux-prd-rh09.abe9.p1`
* `shortname` - Example: `kflux-prd-rh09`
* `cutename` - Example: `rh09`
* `env` - One of `production` or `staging`.
* `network` - One of `public` or `private`.
* `ring` - One of `ring-1`, `ring-2`, `ring-3`, or `ring-4`
* `awsaccount` - The 12-digit AWS account ID for the cluster

4. You are connected to the VPN.

## Procedure

**Run the playbook**, which will prompt you for the seven variables above:

```
❯ ansible-playbook hack/new-cluster/playbook.yaml
```

When the playbook completes, consult the output by inspecting `git diff`.

If satisfied, commit the results, push, and post a pull-request for review by your peers.

Include a description of steps you took to run and verify the automation in the description of your pull request to expedite review.

## Tips

If you do not want to run all steps, but only a subset **you can use tags** to run only tasks tagged with certain tags. For example, if you do not want to verify the vault settings or generate the applicationset changes, but you only want to generate the component overlays, use the `components` tag, like this:

```
❯ ansible-playbook hack/new-cluster/playbook.yaml --tags components
```

If you don't want to specify the variables at prompts, you can **specify variables when invoking the CLI**, like this:

```
❯ ansible-playbook hack/new-cluster/playbook.yaml -e 'cutename=rh09 shortname=kflux-prd-rh09 longname=kflux-prd-rh09.abe9.p1 ring=ring-3 env=production network=public awsaccount=123456789000'
```

If you are **nervous about drift** between the current application manifests and those produced by this automation, you can inspect the different by running this automation and requesting it to produce the config **for an existing cluster**, and then investigate what changes it may have made by looking at `git diff`, like this.

```
❯ ansible-playbook hack/new-cluster/playbook.yaml --skip-tags vault,chains,github -e 'cutename=rh03 shortname=kflux-prd-rh03 longname=kflux-prd-rh03.nnv1.p1 ring=ring-2 env=production network=public awsaccount=123456789000'
❯ git diff
```

The playbook attempts to determine the correct version of some services by inspecting the `main` branch of their git repos and the latest semver tag from quay.io. You can override this by setting commit ids or tag/digest values specifically, like this:

```
❯ ansible-playbook hack/new-cluster/playbook.yaml -e 'commit_id_multi_platform_controller=ec950d0cfb87bcfd6e3a79fc2b5ee40989126123 commit_id_tektoncd_results_for_konflux=425fcd0988b50965139238038e0d3bd3cb4f8bbc commit_id_pipeline_service_exporter=9d2439c8a77d2ce0527cc5aea3fc6561b7671b48 task_runner_tag=3.1.1 task_runner_digest=sha256:790df1bb5ea7a9ce4c1717ff341398ff72c99faed1c2e939a3b4a15ff8f4a493'
```

## Standalone: Create a Tekton Chains signing key

If you only need to create (or verify) a Tekton Chains cosign signing key in Vault — without running the full new-cluster playbook — use `chains-key-playbook.yaml`.

### Prerequisites

* ansible CLI
* vault CLI
* cosign (`go install github.com/sigstore/cosign/v2/cmd/cosign@latest`)
* VPN connection

### Usage

```
❯ ansible-playbook hack/new-cluster/chains-key-playbook.yaml \
    -e 'vault_secret_path=staging/pipeline-service/lightwell-dev/chains-signing-secret'
```

Or interactively (the playbook will prompt for the path):

```
❯ ansible-playbook hack/new-cluster/chains-key-playbook.yaml
```

The `vault_secret_path` is relative to the `stonesoup/` Vault KV mount. The playbook will:

1. Log in to Vault via OIDC
2. Check if the secret already exists (skip if it does)
3. Generate a cosign keypair with a random password
4. Store `cosign.password`, `cosign.key`, and `cosign.pub` in Vault
5. Verify the secret and clean up local files
