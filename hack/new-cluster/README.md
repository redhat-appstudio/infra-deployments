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

4. You are connected to the VPN.

## Procedure

**Run the playbook**, which will prompt you for the five variables above:

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
❯ ansible-playbook hack/new-cluster/playbook.yaml -e 'cutename=rh09 shortname=kflux-prd-rh09 longname=kflux-prd-rh09.abe9.p1 env=production network=public'
```

If you are **nervous about drift** between the current application manifests and those produced by this automation, you can inspect the different by running this automation and requesting it to produce the config **for an existing cluster**, and then investigate what changes it may have made by looking at `git diff`, like this.

```
❯ ansible-playbook hack/new-cluster/playbook.yaml --skip-tags vault,chains,github -e 'cutename=rh03 shortname=kflux-prd-rh03 longname=kflux-prd-rh03.nnv1.p1 env=production network=public'
❯ git diff
```

The playbook attempts to determine the correct version of some services by inspecting the `main` branch of their git repos. You can override this by setting commit ids specifically, like this:

```
❯ ansible-playbook hack/new-cluster/playbook.yaml -e 'commit_id_multi_platform_controller=ec950d0cfb87bcfd6e3a79fc2b5ee40989126123 commit_id_build_definitions=ab6b0b8e40e440158e7288c73aff1cf83a2cc8a9 commit_id_tektoncd_results_for_konflux=425fcd0988b50965139238038e0d3bd3cb4f8bbc commit_id_pipeline_service_exporter=9d2439c8a77d2ce0527cc5aea3fc6561b7671b48'
```
