### Single cmd for Launcher and Driver

This is the combined command line tool for Launcher and Driver. The default image location is `gcr.io/ml-pipeline/kfp-runner`.
Driver is used for capturing inputs, writing to metadata, and triggering Launcher.
Launcher includes running users' custom codes, publishing output to metadata.
The way to run Launcher and Driver is as followed:

```bash
# Under pipelines/backend/src/v2/cmd path
go run ./runner
```

If you want to run launcher:

```bash
# Under pipelines/backend/src/v2/cmd path
go run ./runner launch --copy <runner-volume-path>
```

If you want to run driver:

```bash
# Under pipelines/backend/src/v2/cmd path
go run ./runner drive --type ROOT_DAG
```
