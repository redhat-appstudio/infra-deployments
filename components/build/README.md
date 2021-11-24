# App Studio Build System

The App Studio Build System is composed of the following 
components:

1. OpenShift Pipelines. 
2. AppStudio-specific Pipeline Definitions in `build-templates` for building images.
3. AppStudio-specific `ClusterTasks`.


## Usage
 
As a non-admin user, one would have access to the Pipeline definitions in `build-templates`. 

To be able to use them in one's personal namespace, you can run the following script:

Pipelines will be automatically installed in the namespace when running a build

To validate the pipelines are installed and working, you can run this script which will run a simple single container docker build. 


```
./components/build/hack/test-known-build.sh
```

The above script runs a known docker-build from a sample repository

```
hack/build.sh https://github.com/jduimovich/single-container-app
```

To run any repository with dockerfile in the root of the git repo

```
./components/build/hack/build.sh  <git url>
```
