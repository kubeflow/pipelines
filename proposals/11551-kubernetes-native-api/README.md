# KEP-11551: Introducing a Kubernetes Native API for Pipelines and Pipeline Versions

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Caveats](#caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Migration](#migration)
- [Design Details](#design-details)
  - [Custom Resource Definitions](#custom-resource-definitions)
  - [Go Types](#go-types)
    - [Pipelines](#pipelines)
    - [Pipeline Versions](#pipeline-versions)
    - [Anything Type](#anything-type)
  - [Kubeflow Pipelines API Server](#kubeflow-pipelines-api-server)
    - [Webhooks](#webhooks)
      - [Validating Webhook](#validating-webhook)
      - [Mutating Webhook](#mutating-webhook)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Sync from Kubernetes to the Database](#sync-from-kubernetes-to-the-database)
  - [A Kubernetes Aggregation Layer API Server](#a-kubernetes-aggregation-layer-api-server)
  - [Use Argo Workflows Directly](#use-argo-workflows-directly)
<!-- /toc -->

## Summary

To define a Kubeflow pipeline, the user must interact with the REST API either directly, through the SDK, or through the
Kubeflow dashboard. This is contrary to many other components of Kubeflow, where the Kubernetes API is leveraged as the
declarative API. This also differs from the usual experience of projects that are based on Kubernetes.

The proposal is to allow the user to opt-in to a backwards compatible Kubernetes native API, achieved through custom
resource definitions (CRDs), to define pipelines and pipeline versions. This would enable users to leverage familiar
tools such as Argo CD and the `kubectl` CLI to define pipelines. The proposed approach of using the Kubernetes API as
the storage interface would be backwards compatible in the sense that leveraging the REST API will continue to work as
it does today and changes made through the Kubernetes native API are reflected in the REST API. The same validation and
restrictions would apply to both APIs (e.g. pipeline versions are immutable). This means that existing integrations with
the REST API (e.g. the Kubeflow dashboard) will work seamlessly.

It is the desire to eventually support pipeline runs through a Kubernetes native API, however, this is out of scope for
this feature as there are scale concerns with etcd (e.g. 80k+ runs by some users) and require a different approach with
more tradeoffs.

## Motivation

Users have established GitOps practices with tooling such as Argo CD for Kubernetes configuration management and
application deployments. Allowing users to leverage those tools for Kubeflow pipeline definitions will reduce the
friction felt when adopting the project and provide a more consistent experience across Kubeflow. It also allows for a
familiar and simple query experience from the `kubectl` CLI without needing to maintain a KFP CLI.

### Goals

1. A Kubeflow pipelines user can leverage established GitOps practices to deploy pipelines and pipeline versions through
   a Kubernetes native API.
1. Existing integrations with the REST API (e.g. Kubeflow dashboard) are not impacted.
1. The Kubernetes native API has the same validation and restrictions as the existing REST API (e.g. pipeline versions
   are immutable).
1. The user can opt-in to the Kubernetes native APIs and is not forced to use them or migrate their existing pipeline
   definitions to the Kubernetes native API feature.
1. Provide a sample migration script for users wanting to migrate their existing pipelines and pipeline versions to the
   Kubernetes native API approach.
1. The Kubeflow Pipelines Python SDK is enhanced to allow the output format to be a Kubernetes native API manifest.

### Non-Goals

1. Kubeflow pipeline runs will not be included in the Kubernetes native API at this time. It is a desire to eventually
   do this, but it has scale concerns (e.g. 80k+ runs by some users) that require a different approach with tradeoffs.
1. This will not support the v1 API of Kubeflow pipelines. To continue using the v1 API, the user must continue using
   the default database solution.

## Proposal

The proposal is to allow the user to opt-in to a backwards compatible Kubernetes native API, achieved through custom
resource definitions (CRDs), to define pipelines and pipeline versions. The proposed approach is to use the Kubernetes
API as the pipeline storage interface in the REST API to make things seamless and backwards compatible while not
introducing data duplication or synchronization issues. This is primarily limited to the creation and deletion of
pipelines, but Kubernetes specific metadata is allowed to be updated (e.g. `metadata.labels`).

The same validation and restrictions would apply to both the Kubernetes API and the REST API such as pipeline versions
being immutable, needing to reference an existing pipeline, and have a valid pipeline spec. This will be achieved
through the API server having endpoints for Kubernetes validating and mutating webhooks that leverage the same Go code
used by the API server. This will ensure that there is no version drift of the validation logic between the REST API and
Kubernetes API.

In multiuser mode, the existing RBAC for the REST API performs a subject access review to see if the user has access to
the `pipelines` resource in the `pipelines.kubeflow.org` API group in the Kubernetes namespace. This will be the same
requirement on the Kubernetes native API except that the user will additionally need to have access to the
`pipelineversions` resource. When the Kubernetes native API is enabled on the Kubeflow Pipelines API server, the user
will also need access to the `pipelineversions` resource when leveraging the REST API. When the user has not opted in to
the Kubernetes native API, there is no change.

### User Stories (Optional)

#### Story 1

As a Kubeflow Pipelines user, I can leverage GitOps tool such as Argo CD to define a pipeline and pipeline version.

#### Story 2

As a Kubeflow Pipelines user, I can use either the REST API or Kubernetes API to see available pipelines.

### Notes/Constraints/Caveats (Optional)

#### Caveats

Migrations from storing the pipelines and pipeline versions in the database to the Kubernetes API will lead to the same
pipelines and pipeline versions having different UIDs since the UIDs are generated by Kubernetes. This is acceptable
because there is no guarantee that the user provided the same pipeline spec when recreated through the Kubernetes API,
so having separate UIDs to differentiate them is good for auditability. This is analagous to deleting and reuploading
the same pipeline version today as the UID is randomly generated at upload time.

### Risks and Mitigations

#### Migration

Kubeflow Pipeline administrators must opt-in to the feature and are responsible for migrating the pipelines and pipeline
versions stored in the database to Kubernetes API objects. Although the goal is provide a migration script to help the
user, any migration has its risks. If the migration does not work as expected, the administrator can flip the flag to
use the database again as no data is changed in the database as part of the migration.

## Design Details

### Custom Resource Definitions

The proposal is to add `pipelines.pipelines.kubeflow.org` and `pipelineversions.pipelines.kubeflow.org` custom resource
definitions (CRDs).

Here is an example `Pipeline` manifest:

```yaml
apiVersion: pipelines.kubeflow.org/v2beta1
kind: Pipeline
metadata:
  name: hello-world
  namespace: kubeflow
```

The Kubernetes generated `metadata.uid` field is the pipeline UID that other Kubeflow Pipeline artifacts will refer to.

Here is an example `PipelineVersion` manifest:

```yaml
apiVersion: pipelines.kubeflow.org/v2beta1
kind: PipelineVersion
metadata:
  name: hello-world-v1
  namespace: kubeflow
spec:
  pipelineName: hello-world
  pipelineSpec:
    components:
      comp-generate-text:
        executorLabel: exec-generate-text
        outputDefinitions:
          parameters:
            Output:
              parameterType: STRING
      comp-print-text:
        executorLabel: exec-print-text
        inputDefinitions:
          parameters:
            text:
              parameterType: STRING
    deploymentSpec:
      executors:
        exec-generate-text:
          container:
            args:
              - --executor_input
              - "{{$}}"
              - --function_to_execute
              - generate_text
            command:
              - sh
              - -c
              - "\nif ! [ -x \"$(command -v pip)\" ]; then\n    python3 -m ensurepip ||\
                \ python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1\
                \ python3 -m pip install --quiet --no-warn-script-location 'kfp==2.11.0'\
                \ '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<\"3.9\"' && \"\
                $0\" \"$@\"\n"
              - sh
              - -ec
              - 'program_path=$(mktemp -d)


                printf "%s" "$0" > "$program_path/ephemeral_component.py"

                _KFP_RUNTIME=true python3 -m
                kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"

                '
              - "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import\
                \ *\n\ndef generate_text() -> str:\n    return \"some text from generate_text\"\
                \n\n"
            image: quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0
        exec-print-text:
          container:
            args:
              - --executor_input
              - "{{$}}"
              - --function_to_execute
              - print_text
            command:
              - sh
              - -c
              - "\nif ! [ -x \"$(command -v pip)\" ]; then\n    python3 -m ensurepip ||\
                \ python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1\
                \ python3 -m pip install --quiet --no-warn-script-location 'kfp==2.11.0'\
                \ '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<\"3.9\"' && \"\
                $0\" \"$@\"\n"
              - sh
              - -ec
              - 'program_path=$(mktemp -d)


                printf "%s" "$0" > "$program_path/ephemeral_component.py"

                _KFP_RUNTIME=true python3 -m
                kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"

                '
              - "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import\
                \ *\n\ndef print_text(text: str):\n    print(text)\n\n"
            image: quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0
    pipelineInfo:
      description: A simple intro pipeline
      name: hello-world-v1
    root:
      dag:
        tasks:
          generate-text:
            cachingOptions:
              enableCache: true
            componentRef:
              name: comp-generate-text
            taskInfo:
              name: generate-text
          print-text:
            cachingOptions:
              enableCache: true
            componentRef:
              name: comp-print-text
            dependentTasks:
              - generate-text
            inputs:
              parameters:
                text:
                  taskOutputParameter:
                    outputParameterKey: Output
                    producerTask: generate-text
            taskInfo:
              name: print-text
    schemaVersion: 2.1.0
    sdkVersion: kfp-2.11.0
```

Notice the `spec.pipelineName` refers to the `metadata.name` field of the `Pipeline` object. A Kubernetes validating
webhook will validate that this pipeline exists in the same namespace, and a mutating webhook will add an owner's
reference to the `PipelineVersion` object of the owner `Pipeline`.

The Kubernetes generated `metadata.uid` field is the pipeline version UID that other Kubeflow Pipeline artifacts will
refer to.

The `spec.pipelineSpec` is the YAML of the intermediate representation (IR) that is output from the Kubeflow Pipelines
SDK's compile step. A Kubernetes validating webhook will validate the syntax and ensure that the
`spec.pipelineSpec.pipelineInfo.name` matches the pipeline version's `metadata.name` field.

Additionally, to avoid needing a controller, the CRD leverages a default status of ready. This has the drawback of not
being able to put a timestamp on the status though. A mutating webhook on the status subresource doesn't get executed
when the object is first created, so that was not an option either. See the following example of a created pipeline
version after the mutating webhooks and CRD defaults are applied:

```yaml
apiVersion: pipelines.kubeflow.org/v2beta1
kind: PipelineVersion
metadata:
  creationTimestamp: "2025-01-21T19:38:30Z"
  generation: 1
  labels:
    pipelines.kubeflow.org/pipeline-id: a9b5ce5e-135d-4273-9c8b-0ac5ec1af16d
  name: hello-world-v1
  namespace: kubeflow
  ownerReferences:
    - apiVersion: pipelines.kubeflow.org/v2beta1
      kind: Pipeline
      name: hello-world
      uid: a9b5ce5e-135d-4273-9c8b-0ac5ec1af16d
  resourceVersion: "8856"
  uid: bfa60b22-68a3-444c-9997-df7656b81180
spec:
  pipelineName: hello-world
  pipelineSpec:
    components:
      comp-generate-text:
        executorLabel: exec-generate-text
        outputDefinitions:
          parameters:
            Output:
              parameterType: STRING
      comp-print-text:
        executorLabel: exec-print-text
        inputDefinitions:
          parameters:
            text:
              parameterType: STRING
    deploymentSpec:
      executors:
        exec-generate-text:
          container:
            args:
              - --executor_input
              - "{{$}}"
              - --function_to_execute
              - generate_text
            command:
              - sh
              - -c
              - |2

              if ! [ -x "$(command -v pip)" ]; then
                  python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
              fi

              PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location 'kfp==2.11.0' '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<"3.9"' && "$0" "$@"
              - sh
              - -ec
              - |
                program_path=$(mktemp -d)

                printf "%s" "$0" > "$program_path/ephemeral_component.py"
                _KFP_RUNTIME=true python3 -m kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"
              - |2+

              import kfp
              from kfp import dsl
              from kfp.dsl import *
              from typing import *

              def generate_text() -> str:
                  return "some text from generate_text"

            image: quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0
        exec-print-text:
          container:
            args:
              - --executor_input
              - "{{$}}"
              - --function_to_execute
              - print_text
            command:
              - sh
              - -c
              - |2

              if ! [ -x "$(command -v pip)" ]; then
                  python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
              fi

              PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location 'kfp==2.11.0' '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<"3.9"' && "$0" "$@"
              - sh
              - -ec
              - |
                program_path=$(mktemp -d)

                printf "%s" "$0" > "$program_path/ephemeral_component.py"
                _KFP_RUNTIME=true python3 -m kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"
              - |2+

              import kfp
              from kfp import dsl
              from kfp.dsl import *
              from typing import *

              def print_text(text: str):
                  print(text)

            image: quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0
    pipelineInfo:
      description: A simple intro pipeline
      name: hello-world-v1
    root:
      dag:
        tasks:
          generate-text:
            cachingOptions:
              enableCache: true
            componentRef:
              name: comp-generate-text
            taskInfo:
              name: generate-text
          print-text:
            cachingOptions:
              enableCache: true
            componentRef:
              name: comp-print-text
            dependentTasks:
              - generate-text
            inputs:
              parameters:
                text:
                  taskOutputParameter:
                    outputParameterKey: Output
                    producerTask: generate-text
            taskInfo:
              name: print-text
    schemaVersion: 2.1.0
    sdkVersion: kfp-2.11.0
status:
  conditions:
    - message: READY
      reason: READY
      status: "True"
      type: PipelineVersionStatus
```

### Go Types

#### Pipelines

```go
type PipelineSpec struct {
	Description string `json:"description,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
}
```

#### Pipeline Versions

```go
type PipelineVersionSpec struct {
	Description   string `json:"description,omitempty"`
	CodeSourceURL string `json:"codeSourceURL,omitempty"`
	PipelineName  string `json:"pipelineName,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	PipelineSpec kubernetesapi.Anything `json:"pipelineSpec"`
}

// SimplifiedCondition is a metav1.Condition without lastTransitionTime since the database model doesn't have such
// a concept and it allows a default status in the CRD without a controller setting it.
type SimplifiedCondition struct {
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
	// +kubebuilder:validation:MaxLength=316
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status metav1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Reason string                 `json:"reason" protobuf:"bytes,5,opt,name=reason"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`
}

type PipelineVersionStatus struct {
	Conditions []SimplifiedCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type PipelineVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineVersionSpec   `json:"spec,omitempty"`
	Status PipelineVersionStatus `json:"status,omitempty"`
}
```

#### Anything Type

```go
// Taken from Gatekeeper:
// https://github.com/open-policy-agent/frameworks/blob/6b55861b3fad83f4638ff259ccbd07bff931fd4b/constraint/pkg/core/templates/constrainttemplate_types.go#L130

// Anything is a struct wrapper around a field of type `interface{}`
// that plays nicely with controller-gen
// +kubebuilder:object:generate=false
// +kubebuilder:validation:Type=""
type Anything struct {
	Value interface{} `json:"-"`
}

func (in *Anything) GetValue() interface{} {
	return runtime.DeepCopyJSONValue(in.Value)
}

func (in *Anything) UnmarshalJSON(val []byte) error {
	if bytes.Equal(val, []byte("null")) {
		return nil
	}
	return json.Unmarshal(val, &in.Value)
}

// MarshalJSON should be implemented against a value
// per http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
// credit to K8s api machinery's RawExtension for finding this.
func (in Anything) MarshalJSON() ([]byte, error) {
	if in.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(in.Value)
}

func (in *Anything) DeepCopy() *Anything {
	if in == nil {
		return nil
	}

	return &Anything{Value: runtime.DeepCopyJSONValue(in.Value)}
}

func (in *Anything) DeepCopyInto(out *Anything) {
	*out = *in

	if in.Value != nil {
		out.Value = runtime.DeepCopyJSONValue(in.Value)
	}
}
```

### Kubeflow Pipelines API Server

The Kubeflow Pipelines API server will have a new flag of `--pipelines-store` with the default option of `database` and
another option of `kubernetes`. When set to `database`, the existing behavior of storing pipelines in the database
stays. When set to `kubernetes`, the pipelines store will leverage the Kubernetes API, essentially acting as a proxy for
backwards compatibility. When set to the `kubernetes` mode and the API server is in multiuser mode, queries for pipeline
versions should check the `pipelineversions.pipelines.kubeflow.org` resource rather than just the
`pipelines.pipelines.kubeflow.org` resource.

The Kubernetes API does not support server-side sorting and field selectors are very limited until
[v1.31+ CRD selectable fields support](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#crd-selectable-fields).
The REST API will need to perform client-side filtering before returning the result to keep the same experience. The
returned next page token should provide enough information to know where the client side filtering left off.

#### Webhooks

As noted in the previous section, the API server will have two new endpoints:

- `/webhooks/validate-pipelineversion` - for a validating webhook for pipeline versions.
- `/webhooks/mutate-pipelineversion` - for a mutating webhook for pipeline versions.

If the cluster has multiple installations of Kubeflow Pipelines (e.g. multiple standalone), administrator has two
options:

- The webhook configuration could have a namespace selector set at `webhooks[0].namespaceSelector` to target that
  specific installation. The administrator can create multiple webhook entries in the same
  `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects or create one object per installation.
- A new option in the API server of `--global-kubernetes-webhook-mode` can be enabled to run the API server with
  **only** the Kubernetes webhook endpoints enabled. The administrator would have just one entry in the
  `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects that point to this global instance. The
  administrator should ensure that all Kubeflow Pipeline servers on the cluster are the same version in this case.

##### Validating Webhook

The validating webhook will perform the following checks on **creation** of `PipelineVersion` objects:

- Verify the `spec.pipelineName` matches an existing `Pipeline` object in the same namespace.
- Verify the `spec.pipelineSpec.pipelineInformation.name` field matches the `metadata.name` field.
- Verify the `spec.pipelineSpec` is syntactically valid leveraging the
  `github.com/kubeflow/pipelines/backend/src/apiserver/template` package.

The validating webhook will perform the following checks on **updates** of `PipelineVersion` objects:

- Verify that `spec` did not change. In other words, allow metadata changes such as labels and annotations only. This
  may be as simple as checking if the `metadata.generation` changed but more research is needed.

##### Mutating Webhook

The mutating webhook will perform the following mutations on **creation** and **updates** of `PipelineVersion` objects:

- Set an owner's reference to the `Pipeline` object referenced in `spec.pipelineName`.
- Set a label of `pipelines.kubeflow.org/pipeline-id` to the pipeline's UID for convenience.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid
enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

#### Unit Tests

The unit tests will be used to cover logic such as the validating and mutating webhooks and conversion from Kubernetes
API representation to REST API representation and the reverse. The remainder shall be covered through integration tests
where a real Kubernetes cluster is present.

#### Integration tests

- Add an additional matrix to the GitHub workflows `API integration tests v2`, `Frontend Integration Tests`, and
  `basic-sample-tests` to test with the Kubernetes API being the storage for piplines. This ensures backwards
  compatibility through the REST API.
- Add an additional end to end test that leverages a sample pipeline that compiles out to the Kubernetes manifest format
  using the Python SDK and submits it directly against the Kubernetes API with `kubectl`. Then a pipeline run is started
  leveraging this pipeline version.

### Graduation Criteria

<!--
Clearly define what it means for the feature to be implemented and
considered stable.
If the feature you are introducing has high complexity, consider adding graduation
milestones with these graduation criteria:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]
[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
-->

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubeflow Pipelines and Kubeflow release where an initial version of the KEP was available
- the version of Kubeflow Pipelines and Kubeflow where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

The main drawbacks are:

- The Kubernetes API does not support server-side sorting and field selectors are very limited until
  [v1.31+ CRD selectable fields support](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#crd-selectable-fields).
  The REST API will need to perform client-side filtering before returning the result to keep the same experience. This
  is inefficient and may have performance issues in a very large environments.
- Queries for pipelines through the REST API will be slower since it must query the Kubernetes API rather than issue a
  faster SQL query.
- Backups and restores are less straight forward since you need to backup the pipeline versions stored in Kubernetes'
  etcd as well.

## Alternatives

### Sync from Kubernetes to the Database

It was considered to have a controller that watches for `Pipeline` and `PipelineVersion` objects and relays the request
to the REST API. The benefits to that solution are:

- The REST API code requires less code changes, though the validation webhook endpoint would still be needed to get
  immediate feedback on the validity of the pipeline spec and to enforce immutability of the pipeline version.
- A backup and restore scenario is a little bit easier since you only need to restore the database backup and not the
  Kubernetes objects stored in etcd.
- REST API list queries are more efficient since the sorting and filtering can be done at the SQL level rather than
  client-side on the REST API.

The downsides to that solution are:

- Data synchronization issues. An example scenario is deleting a pipeline version through the REST API does not delete
  it in Kubernetes, so it would get constantly recreated by the controller.
- Data synchronization delays. An example is a user defines a pipeline version in the Kubernetes API and then tries to
  create a run, but the pipeline version hasn't been synced through the REST API yet, so creating the run would fail.
- The user querying the Kubernetes API may see an incomplete list of pipeline versions depending on if the pipeline
  version was defined in the REST API or through the Kubernetes API. One workaround would be for the controller to
  constantly poll the REST API to make sure the two are in sync. Another workaround is for the REST API to also store
  the data through the Kubernetes API. The former workaround could be resource intensive, and the latter could lead to
  things being out of sync. An example scenario is the API server fails to make the Kubernetes API request (transient
  error, webhook misconfiguration, and etc.). Another example scenario is the API server restarts before it completes
  the Kubernetes API request. Another example could be conflicts such as the pipeline version already exists in
  Kubernetes.

In the end, the downsides outweighed the tradeoffs in the proposed solution of using the Kubernetes API as the pipelines
store.

### A Kubernetes Aggregation Layer API Server

One alternative is to register the Kubeflow Pipelines REST API server as a Kubernetes aggregation layer API service.
This would allow for Kubernetes requests to be sent to the Kubeflow Pipelines REST API for certain resources (e.g.
`*.pipelines.kubeflow.org`).

The benefits to this solution are:

- Like the preferred proposal, there are no data synchronization issues since the requests are forwarded to the REST API
  directly.
- The data would continue to be stored in the database so backup and restore situations would be simpler, and queries
  could efficiently use SQL as the implementation.

The downsides are:

- Complex to implement. There isn't a well maintained library to do to this.
- Not a common pattern and so it could be difficult to maintain.
- Cannot provide native Kubernetes API conventions such as watch requests since there is no concept of watches or
  resource versions in Kubeflow Pipelines. This could be problematic for GitOps tooling that depends on watch requests
  to detect diffs and status changes.
- There is no concept of namespace selectors when registering an aggregated API server, so multiple installations of
  standlone Kubeflow Pipelines on the same cluster would be tricky as it would require a global reverse proxy to forward
  the request to the proper Kubeflow Pipelines REST API.
- The aggregated API server needs to validate the request came from the Kubernetes API server and it needs to handle
  authorization.

### Use Argo Workflows Directly

Argo Workflows is already Kubernetes native and is the primary pipeline engine in the Kubeflow Pipelines, however, the
v2 design of Kubeflow Pipelines is pipeline engine agnostic. Additionally, the current `Workflow` objects produced by
Kubeflow Pipelines are not an API and include many implementation details that are difficult for a user to understand
and thus review in the a GitOps workflow.
