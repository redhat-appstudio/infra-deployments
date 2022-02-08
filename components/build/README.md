
# App Studio Build System

The App Studio Build System is composed of the following components:

1. OpenShift Pipelines. 
2. AppStudio-specific Pipeline Definitions in `build-templates` for building images.
3. AppStudio-specific `ClusterTasks`.
4. Tekton Chains.

This repository installs all the components and includes a set of example scripts that simplify usage and provide examples of a working system. There are no additiona components needed to use the build system API, howvever some utilities and scripts are provided to demonstrate functionality. 

## Quickstart 

To try out a pre-configured, follow these steps. 

| Steps    |    |
| ----------- | ----------- | 
| 1.  Create project for your pipelines execution. This can be run as any non-admin user (or admin)  and is needed to hold your execution pipelines.  |  oc new-project demo     |  
| 2.  Run build-deploy example with a quarkus app. |  ./hack/build/build-deploy.sh  https://github.com/devfile-samples/devfile-sample-code-with-quarkus
| 3.  View your build on the OpenShift Console under the pipelines page or view the logs via CLI. | `./hack/build/ls-builds.sh` or  `tkn.exe pipelinerun logs`      |

## Usage
 
A sample script `build.sh` is provided which uses the App Studio Build Service API to demonstrate launching a build and inspecting the results.
As a proof-of-concept, an optional `build-deploy.sh` script is included to take the build image and run it. . 

```
hack/build/build.sh  git-repo-url <optional-pipeline-name>
```
also the equivalent build but with an associated deploy. 
```
hack/build/build-deploy.sh  git-repo-url <optional-pipeline-name>
```

The `git-repo-url` is the git repository with your source code.
The `<optional-pipeline-name>` is the name of one of the pipelines documented in the App Studio API Contract <url here>. This pipeline name can be provide when the automatic build type detection does not find a supported build type. 
Note: Normally the build type would be done automatically by (by the Component Detection Query) which maps devfile or other markers to a type of build needed. The build currently uses a shim `repo-to-pipeline.sh` to map file markers to a pipeline type. For testing and experiments the  `optional-pipeline-name`  parameter can override the default pipeline name. 

The current build types supported are: `devfile-build, `docker-build`, `java-buider` and `node-js-builder`.

For a quick "do nothing pipeline" run you can specify the `noop` buider and have a quick pipeline run that does nothing except print some logs. 

`.hack/build/build.sh  https://github.com/jduimovich/single-container-app  noop`

Pipelines will be automatically installed when running a build via an OCI bundle mechanism. 

To see what builds you have run, use the following examples.

Use `.hack/build/ls-builds.sh` to show all builds in the system, and `.hack/build/ls-builds.sh <build-name>` to get the stats for a specific build.

## Testing 

To validate the pipelines are installed and working, you can run `./hack/build/m2-builds`  script which will build all the samples planned for milestone 2. 

To deploy all the builds as they complete, add the `-deploy` option.
```
./hack/build/m2-builds -deploy

```
You can also run the noop build `hack/build/quick-noop-build.sh`, that executes in couple seconds to validate a working install.
 
## Other Build utilties 

The build type is identified via temporary hack until the Component Detection Query is available which maps files in your git repo to known build types. See `hack/build/repo-to-pipeline.sh`  which will print the repo name and computed builder type.

The system will fill with builds and logs so a utility is provided to prune pipelines and cleanup the associated storage. This is for dev mode only and will be done autatically by App Studio builds.
Use `hack/build/prune-builds.sh` for a single cleanup pass, and `hack/build/prune-builds-loop.sh` to run a continous loop to cleanup extra resources. 

Use `./hack/build/util/check-repo.sh` to test your what auto-detect build will return.  
```
./hack/build/check-repo.sh  https://github.com/jduimovich/single-java-app
https://github.com/jduimovich/single-java-app   -> java-builder
  
```
If you want to check all your repos to see which ones may build you can use this script. You need to set you github id `export MY_GITHUB_USER=your-username` and it will test your repo for buildable content.  

```
./hack/build/ls-all-my-repos.sh | xargs -n 1 ./hack/build/check-repo.sh
```