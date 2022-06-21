# Samples for compiler unit tests.

To update pipeline golden snapshots:

```bash
for f in sdk/python/kfp/compiler/test_data/pipelines/*.py ; do echo "$f" && python3 "$f" ; done
```


To update component golden snapshots:
```bash
for f in sdk/python/kfp/compiler/test_data/components/*.py ; do echo "$f" && python3 "$f" ; done
```
