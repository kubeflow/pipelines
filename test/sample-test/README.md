# sample-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies:

1. (optional) edit [requirements.in](requirements.in)
1. run

    ```bash
    ./hack/update_requirements.sh
    ```

    to update and pin the transitive dependencies.

Some dependencies are resolved at the time running this command, so without editing
requirements.in, the result will still change over time.
