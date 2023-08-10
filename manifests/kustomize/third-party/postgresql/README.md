## Build PostgreSQL yaml

```bash
# In this folder of manifests/kustomize/third-party/postgresql
rm -rf build
mkdir build
kustomize build ./base -o build
```

## Deploy PostgreSQL container

```bash
# In this folder of manifests/kustomize/third-party/postgresql
kubectl -n <your_namespace> apply -f build
```