# go test ./... -namespace kubeflow -args -runIntegrationTests=true
go test ./... -v -namespace kubeflow -args -runUpgradeTests=true -testify.m=$1
