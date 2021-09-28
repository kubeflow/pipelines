Building, testing, releasing containers for Metrics Evaluation Backend. gcloud,
docker and virtualenv required throughout.

Head depenendcies in google3/third_party

*   e2e_local_test.sh: Test head code with head depenencies. This checks the
    present state of the code against dependencies found in google3/third_party
    which are the HEAD versions of those libraries. It is possible these are
    unreleased versions of the libraries found in PYPI. A successful run
    results in a TEST SUCCESS message.

Dependencies in requirements.txt

NOTE: you may need to run `gcloud auth application-default login` before
executing the following:

*   local_direct.sh: Build and run an example metrics set directly in process.

*   local_dataflow.sh: Build and run an example metrics set with DataFlow
    backend.

*   container_presubmit.sh: Build a container and test it locally.

*   container_candidate.sh: Build and push a container "candidate" and test
    locally.

*   dataflow_test.sh: Run dataflow-flex from the candidate container.

*   dataflow_release.sh: Mark the public template for the "latest" container.

*   Manually tag the candidate as "latest" to make it live for production.

*   container_test.sh: Test the container, optionally specifying a tag.
