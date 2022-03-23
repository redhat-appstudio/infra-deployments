Enterprise Contract Experiments with OPA and Rego
=================================================

Usage
-----

Assuming you're logged in to a cluster with at least one pipeline run:

    ./fetch-data.sh <pipelinerun-name>
    ./validate.sh

If you don't specify a pipeline run it will use the most recent.

If you don't have a pipeline run and a cluster handy you can test this with
some canned data.

    ./fetch-canned-test-data.sh
    ./validate.sh

The output is pretty boring when everything passes, so to see some failures,
manually edit the yaml under ./data, then rerun ./validate.sh. Look at the
policies to get some ideas about how you can invalidate the pipeline run data.

Example output showing some validation failures:

    $ ./validate.sh
    [
      {
        "msg": "Not all taskruns have 'chains.tekton.dev/transparency' and 'chains.tekton.dev/signed' annotations set"
      },
      {
        "msg": "Not all taskruns have the same transparency log server!"
      },
      {
        "msg": "Taskrun nodejs-builder-2022-03-23-155407-appstudio-configure-buil-6rpx9 has signed status of 'nope'"
      }
    ]

To do
-----

* More useful and realistic policies such as "particular taskrun was present"
* Run the whole thing in a task and expose the validation results in the task output
* Pull down data from other sources, e.g. tekton results or an image registry
* More useful output to explain the reasons for failures
* Incorporate existing cosign verify task
* Less clunky rego
* Can/should we use [conftest](https://www.conftest.dev/)?

See also
--------

* https://github.com/dirgim/hacbs-conftest/
