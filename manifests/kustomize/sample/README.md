# Kustomize sample for a customized installation

1. Customize your values
- Edit **params.env** to your values
- Edit kustomization.yaml to set your namespace

2. Install your setup

```
kubectl apply -k sample/
```