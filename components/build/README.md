# App Studio Build System

The App Studio Build System is composed of the following 
components:

1. OpenShift Pipelines. 
2. AppStudio-specific Pipeline Definitions in `build-templates` for building images.
3. AppStudio-specific `ClusterTasks`.


## Usage
 
As a non-admin user, one would have access to the Pipeline definitions in `build-templates`. 

To be able to use them in one's personal namespace, you can run the following script:

```
./components/build/hack/install-pipelines.sh
```
This will install the default set of build pipelines into your namespace. 

To validate the pipelines are installed and working, you can run this script which will run a simple single container docker build. 
```
./components/build/hack/test-known-build.sh
```

To run any repository with dockerfile in the root of the git repo

```
./components/build/hack/build.sh  <git url>
```
