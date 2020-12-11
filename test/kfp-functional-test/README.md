# kfp-functional-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies:
1. edit [requirements.in](requirements.in)
1. run
    ```
    ../../backend/update_requirements.sh <requirements.in >requirements.txt
    ```
    to update and pin the transitive dependencies.
