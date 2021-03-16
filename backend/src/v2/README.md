# V2 Backend POC

* You can run this POC on a KFP cluster, take a look at `./compiler`.
* You can run e2e testing for this POC locally by:

    First, in one terminal, run the following to start a proxy to MLMD service in your KFP cluster (do not use your production cluster!):

    ```bash
    make proxy
    ```

    Then, run the e2e testing:

    ```bash
    make
    ```
