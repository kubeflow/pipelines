# Files Changed in PR - Review List

## ✅ Core Feature Files (Pipeline Run Parallelism)
- `api/v2alpha1/pipeline_spec.proto` - Added pipeline_run_parallelism field
- `api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go` - Generated Go code
- `sdk/python/kfp/dsl/pipeline_config.py` - SDK PipelineConfig changes
- `sdk/python/kfp/compiler/pipeline_spec_builder.py` - Compiler changes
- `backend/src/apiserver/resource/resource_manager.go` - Main implementation
- `backend/src/apiserver/resource/resource_manager_test.go` - Unit tests
- `backend/src/v2/compiler/argocompiler/argo.go` - Argo compiler changes
- `README.md` - Documentation update

## ✅ Test Files
- `sdk/python/kfp/compiler/compiler_test.py` - Compiler tests
- `sdk/python/test/compilation/pipeline_compilation_test.py` - Compilation tests
- `backend/test/end2end/pipeline_e2e_test.go` - E2E test integration
- `backend/test/end2end/utils/e2e_utils.go` - E2E test utilities

## ✅ Generated/Test Data Files
- `test_data/compiled-workflows/pipeline_with_run_parallelism.yaml` - Golden file (NEW)
- `test_data/sdk_compiled_pipelines/valid/essential/pipeline_with_run_parallelism.py` - Test pipeline (NEW)
- `test_data/sdk_compiled_pipelines/valid/essential/pipeline_with_run_parallelism.yaml` - Test pipeline YAML (NEW)
- `backend/test/proto_tests/testdata/generated-1791485/pipeline_spec.pb` - Generated protobuf
- `backend/test/proto_tests/testdata/generated-1791485/pipeline_version.pb` - Generated protobuf
- `backend/test/proto_tests/testdata/generated-1791485/run_completed_with_spec.pb` - Generated protobuf

## ✅ Kubernetes/Manifest Files
- `manifests/kustomize/base/pipeline/cluster-scoped/ml-pipeline-configmap-manager-clusterrole.yaml` - NEW
- `manifests/kustomize/base/pipeline/cluster-scoped/ml-pipeline-configmap-manager-clusterrolebinding.yaml` - NEW
- `manifests/kustomize/base/pipeline/cluster-scoped/params.yaml` - NEW
- `manifests/kustomize/base/pipeline/cluster-scoped/kustomization.yaml` - Updated
- `manifests/kustomize/base/pipeline/ml-pipeline-apiserver-role.yaml` - Updated

## ✅ Kubernetes Client Files
- `backend/src/apiserver/client/kubernetes_core.go` - ConfigMap client methods
- `backend/src/apiserver/client/kubernetes_core_fake.go` - Fake client for tests
- `backend/src/cache/client/kubernetes_core.go` - Cache client methods
- `backend/src/cache/client/kubernetes_core_fake.go` - Fake cache client

## ⚠️ Files from Master Merge (Review if needed)
- `test_data/compiled-workflows/pipeline_with_workspace.yaml` - SDK version bump (2.14.3 → 2.14.6)
- `test_data/sdk_compiled_pipelines/valid/critical/pipeline_with_workspace.py` - SDK version bump
- `test_data/sdk_compiled_pipelines/valid/critical/pipeline_with_workspace.yaml` - SDK version bump
- `.github/actions/create-cluster/action.yml` - CI changes from master
- `.github/actions/github-disk-cleanup/action.yml` - NEW (from master)
- `.github/actions/github-disk-cleanup/free-disk-space.sh` - NEW (from master)
- `.github/workflows/api-server-tests.yml` - CI workflow changes
- `.github/workflows/e2e-test.yml` - CI workflow changes
- `.github/workflows/kfp-kubernetes-native-migration-tests.yaml` - CI workflow changes
- `.github/workflows/kfp-sdk-client-tests.yml` - CI workflow changes
- `.github/workflows/legacy-v2-api-integration-tests.yml` - CI workflow changes
- `.github/workflows/upgrade-test.yml` - CI workflow changes
- `go.mod` - Dependency updates from master
- `go.sum` - Dependency updates from master
- `backend/src/cache/deployer/deploy-cache-service.sh` - kubectl download fix (your change)

## ❌ Files to Remove/Revert
- `proposals/11875-pipeline-workspace/README.md` - KEP file (formatting change, should be reverted)

## Summary
- **Total files:** 43
- **Core feature files:** 8
- **Test files:** 4
- **Generated/test data:** 6
- **Kubernetes/manifests:** 5
- **Kubernetes client:** 4
- **From master merge:** 15
- **To revert:** 1 (KEP file)

