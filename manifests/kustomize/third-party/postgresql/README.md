## Build PostgreSQL yaml

```bash
# In this folder of manifests/kustomize/third-party/postgresql
rm -rf build
mkdir buidl
kustomize build ./base -o build
```

## Deploy PostgreSQL container

```bash
# In this folder of manifests/kustomize/third-party/postgresql
kubectl apply -f build
```