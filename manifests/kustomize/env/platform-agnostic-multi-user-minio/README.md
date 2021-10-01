1. Prepare S3
Create S3 bucket on prem

2. Customize your values
- Edit [params.env](params.env), [secret.env](secret.env). 
- (Optional) Put SSL cert in files/ssl-cert.crt and uncomment in kustomizaion.yaml

3. Install
`./kustomize build pipelines/manifests/kustomize/env/platform-agnostic-multi-user-mini | kubectl apply -f -`
