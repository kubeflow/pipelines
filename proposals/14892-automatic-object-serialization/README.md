# KEP-14892: Automatic Typed Object Serialization

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Supported Types and Formats](#supported-types-and-formats)
  - [Contracts and Dependencies](#contracts-and-dependencies)
  - [Artifact Bundle and Runtime](#artifact-bundle-and-runtime)
  - [Sub-DAGs, Control Flow, and Caching](#sub-dags-control-flow-and-caching)
  - [Security and Error Reporting](#security-and-error-reporting)
  - [Frontend Considerations](#frontend-considerations)
  - [KFP Local Considerations](#kfp-local-considerations)
  - [Migration Strategy](#migration-strategy)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [E2E tests](#e2e-tests)
    - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Open Questions](#open-questions)
- [References](#references)
<!-- /toc -->

## Summary

Let Python components receive and return supported DataFrames and trained models
using native type annotations. KFP selects a serializer, persists an ordinary
artifact, and reconstructs it in the consuming component. Sub-pipelines forward
references, not live objects. This adds automatic serialization to existing
artifact storage, not a shared-memory service or a new object store.

## Motivation

Authors repeatedly write save/load code, while `Dataset` and `Model` annotations
alone do not identify a file format or reconstruction type. Typed serialization
would make component and sub-pipeline interfaces reusable without this boilerplate.
Related demand appears in [#6304](https://github.com/kubeflow/pipelines/issues/6304),
which was closed for inactivity.

### Goals

- Pass native objects between components and through nested sub-pipelines.
- Support multiple DataFrame libraries/formats and framework-specific models.
- Preserve artifact lineage, caching, local execution, and explicit path access.
- Allow explicit application adapters without making every framework a core
  SDK dependency.

### Non-Goals

- Shared mutable objects, zero-copy guarantees, or a new storage service.
- Arbitrary Python objects, implicit pickle, or complete process/training state.
- Implicit library conversion, model export, or collection of distributed/lazy
  DataFrames.
- Replacing existing parameter/artifact APIs or adding rich artifact viewers.

## Proposal

Introduce opt-in, versioned serialization contracts on artifact ports. The
compiler resolves the native type, writer format/options, and permitted readers.
At runtime, the executor loads inputs before calling the component and saves
returned objects afterward. Existing scalar, JSON, and Pydantic parameters are
unchanged.

### SDK User Experience

Illustrative API, not current SDK support. Component images must provide the
chosen frameworks and compatible KFP/adapters; image configuration is omitted.
Frameworks must also be importable when authoring annotated Python functions.

```python
import pandas as pd
from kfp import dsl
from xgboost import XGBClassifier


@dsl.component
def prepare() -> pd.DataFrame:
    """Produce an artifact inside the pipeline."""
    import pandas as pd

    return pd.DataFrame({"feature": [0.0, 1.0, 2.0, 3.0], "label": [0, 0, 1, 1]})


@dsl.component
def train(data: pd.DataFrame) -> XGBClassifier:
    """Train without managing artifact files."""
    from xgboost import XGBClassifier

    return XGBClassifier(n_estimators=20, max_depth=3).fit(
        data.drop(columns=["label"]), data["label"]
    )


@dsl.component
def predict(data: pd.DataFrame, model: XGBClassifier) -> pd.DataFrame:
    """Use a reconstructed model."""
    return data.assign(prediction=model.predict(data.drop(columns=["label"])))


@dsl.pipeline
def training_stage(data: pd.DataFrame) -> XGBClassifier:
    """Forward a model through a sub-pipeline."""
    return train(data=data).output


@dsl.pipeline
def object_pipeline() -> pd.DataFrame:
    """Connect artifact-backed interfaces."""
    data = prepare().output
    model = training_stage(data=data)
    return predict(data=data, model=model.output).output
```

Pipeline bodies operate on channels, not DataFrame/model instances. Remote run
submission cannot bind root artifact inputs; extending that API is out of scope.
Reject such typed root inputs at compilation. Produce artifacts in the graph or
import them using a URI parameter.

An optional `Annotated` marker selects output encoding; its spelling is open:

```python
from typing import Annotated

import polars as pl

PandasParquet = Annotated[pd.DataFrame, dsl.Serialization(format="parquet")]
PolarsArrow = Annotated[pl.DataFrame, dsl.Serialization(format="arrow-ipc")]
```

Default encodings are fixed on leaf outputs at compilation. Bare input and
sub-pipeline annotations constrain the Python type, not an encoding; forwarding
ports inherit the producer's codec. Only an explicit `Serialization(...)`
imposes a format constraint. Annotations cannot transcode artifacts.
Named output fields remain separate outputs.

### User Stories

Authors can prepare a DataFrame, train a model inside a reusable sub-pipeline,
and consume that model for prediction without writing artifact IO. A deployment
component can still request `Input[Model]` and inspect the bundle without loading
the model.

### Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Silent dtype/state loss | Publish round-trip guarantees; reject unsupported values. |
| Unsafe reconstruction | Trusted readers, restricted model loading, and bundle validation. |
| Memory and IO surprises | Explicit eager-loading opt-in, resource limits, and path-based escape hatches. |
| Dependency drift or stale cache hits | Versioned contracts, reader compatibility, and contract-aware cache identity. |
| Expanding maintenance burden | Bounded class/version support and optional dependencies. |

SDK/runtime and security reviewers should review the initial adapters before
alpha; frontend/local maintainers should review their respective behavior.
Reviewers and approvals are not assigned by this proposal.

## Design Details

### Supported Types and Formats

Support is limited to documented classes, dtypes, options, and library versions.

| Runtime type | Artifact | Encoding | Target |
| --- | --- | --- | --- |
| Pandas / Polars DataFrame; PyArrow Table | `Dataset` | Parquet default; Arrow IPC selectable | Alpha |
| Supported scikit-learn estimators | `Model` | `skops` with an explicit trusted-type policy | Alpha |
| XGBoost Booster, XGBClassifier, XGBRegressor | `Model` | Native UBJSON default; JSON selectable | Alpha |
| Non-object NumPy ndarray | `Artifact` | NPY, pickle disabled | Beta |
| Supported Keras models | `Model` | Native `.keras`, restricted loading | Beta |
| Registered PyTorch models | `Model` | Tensor weights plus explicit architecture/configuration | Beta |
| Supported Hugging Face Transformers models (PyTorch) | `Model` | `save_pretrained` configuration and Safetensors weights/shards | Beta |
| Application types | Adapter-declared | Explicit registered codec | Extension API |

DataFrame adapters must specify column/index, null, precision, categorical,
timezone, and extension-dtype guarantees. Unsupported values fail rather than
silently stringify or coerce. Pandas `object` columns containing supported scalar
values need not be rejected. Cross-library conversion stays explicit. CSV,
distributed DataFrames, and lazy collection remain explicit artifact workflows.

Model adapters declare preserved state and reconstruction requirements:

- XGBoost uses `save_model`/`load_model`, not `dump_model` or memory snapshots.
  Distinguish Booster from sklearn wrappers, preserve supported prediction and
  feature metadata, and validate reader compatibility. Wrappers use the XGBoost
  adapter, not the generic sklearn adapter. Native IO omits arbitrary Python
  objectives, metrics, callbacks, and wrapper state; reject unsupported state
  or require an explicit application adapter.
- Sklearn loading must not trust every type discovered in a file. Keras custom
  objects and PyTorch architecture constructors must be explicitly allowed,
  preinstalled code. Weights alone do not reconstruct an arbitrary model.
- For transformer models, start with Hugging Face Transformers on PyTorch.
  Supported `PreTrainedModel` subclasses use a dedicated adapter, not the generic
  PyTorch adapter: save configuration, Safetensors weights, all shards/indexes,
  and generation configuration when applicable. Reconstruct the approved class
  offline with `from_pretrained`, requiring Safetensors and disabling remote
  code and artifact-selected kernels. Preserve supported tensor dtypes/tied
  weights; load in evaluation mode, with device placement chosen by the consumer.
  Tokenizers/processors are separate explicit artifacts. Adapter-only
  (PEFT/LoRA), quantized, and distributed checkpoints require later explicit
  support, not implicit base-model downloads or pickle fallback.
- Optimizer/resumption state must be explicitly included or excluded by each
  adapter. GPU allocations and process resources are never restored. Native
  formats do not guarantee that hostile files are safe.

### Contracts and Dependencies

Add optional port contracts to the pipeline IR: logical type, codec ID/revision,
format/options, and permitted reconstruction bindings. Preserve these through
component loading, imports, and sub-DAG interfaces. Artifact metadata alone
cannot describe an independently loaded consumer's input contract.

Resolve deterministic adapter registrations into canonical executor bindings
emitted in container arguments, so normal remote/local fingerprints include
them. Ambiguity requires an explicit choice. The compiler checks known
producer/consumer compatibility; the executor checks the actual manifest,
schema, dependencies, and reconstructed type.

Adapters/frameworks come from the component image or explicitly declared
dependencies, never artifact-directed installation. Load optional dependencies
lazily and support KFP's `--no-deps` runtime bootstrap. Generated lightweight
components must preserve imports/bindings without relying on notebook globals.
Custom adapter code remains application code.

Contract-bearing pipelines require a new schema version accepted by upgraded
backends; pipelines without contracts retain the existing version. Use backend
schema-version validation to reject unsupported pipelines, not ignored optional
fields. Exact fields/version require API review; regenerated bindings,
compiler/driver/executor validation, and documented minimum versions gate alpha.

### Artifact Bundle and Runtime

Each object is one directory artifact:

```text
<artifact.path>/
  manifest.json
  payload/...
```

The manifest contains the contract, producer versions, required schema/state
metadata, and payload paths/sizes/digests. Keep a bounded type/codec/format summary
in artifact metadata; large schemas and inventories stay in the bundle.
Preserve unrelated user metadata.

1. The launcher or local runner supplies normal artifact/workspace paths.
2. The executor validates inputs and reconstructs objects with permitted readers.
3. Outputs are serialized into attempt-local staging; write the final manifest
   only after payload writes succeed.
4. Normal upload/registration publishes the completed artifact. Required
   persistence must finish before the task succeeds.

This does not require atomic multi-file object-store uploads: successful task
publication is the visibility boundary. Retries must not publish partial bundles
or overwrite existing immutable outputs. Workspaces do not remove durability
requirements. Explicit artifact consumers receive the bundle directory, not a
guaranteed single native file; serving integrations may need an export adapter.

Initially, typed bundles use object-store artifacts and compatible workspaces.
`oci://`/Modelcar artifacts retain explicit artifact handling; automatic
serialization for them is out of scope. Object-store inputs stage the complete
bundle on local disk before reconstruction. Use explicit artifact/path APIs
when data exceeds disk/memory budgets or requires partial loading.

### Sub-DAGs, Control Flow, and Caching

Sub-DAGs forward references/contracts without serialization or transfer. Fan-out
consumers reconstruct independently; writable shared mappings must not mutate
published inputs. Conditions and optional artifacts retain existing semantics;
unsupported output forms fail rather than fabricate a serialized `None`.
Alpha uses existing explicit artifact-list interfaces for `dsl.Collected`.
Automatic `list[T]` reconstruction targets beta, after collection/resource
limits are defined. It must not concatenate DataFrames or replace JSON lists.

Imported bundles require a declared compatible contract and the same validation.
Legacy files need explicit adoption/conversion, not a format guessed from their
extension.

Both producer and consumer cache identities include resolved codec revisions,
options, and reader/custom-adapter bindings, alongside normal code/image/inputs.
Custom keys currently bypass that normal identity: change both remote and local
custom-key paths to also hash the contract fingerprint for typed ports. Existing
untyped custom-key behavior stays unchanged.

Caches store references, not objects. Validate reused bundles before loading;
missing/incompatible artifacts follow an explicit failure or documented cache
miss path, never a permissive loader fallback. Changed defaults cannot
reinterpret old compiled contracts. Representation migrations create new
artifacts, not rewritten cached outputs.

### Security and Error Reporting

Treat manifests and payloads, including imports/cache hits, as untrusted input.
Reuse KFP's existing security boundaries rather than add a security service.

- No implicit pickle/cloudpickle or unrestricted joblib. Reader code and allowed
  types come from trusted component bindings, not artifact imports, package
  requests, or artifact-supplied allowlists.
- Validate versions, paths, sizes, and digests. Reject traversal, absolute paths,
  escaping links, and payload references that trigger arbitrary remote downloads.
  Digests detect corruption, not authenticity if both manifest and payload change.
- Maintain supported decoder dependencies; native formats can still have parser
  vulnerabilities. Bound manifest/schema parsing and account for file counts,
  decompression, and memory amplification alongside task CPU/memory/time limits.
- Reconstruction runs only in the consuming component, not the compiler/API/UI.
  Custom loaders inherit component privileges; local `SubprocessRunner` inherits
  the user's permissions and is not a sandbox.
- Preserve namespace/store/workspace access restrictions for imports and cache
  hits without elevated privileges. Protect manifests/staging like payloads,
  including configured encryption, retention, and temporary-file cleanup.
- Errors identify port, type, codec, and corrective action, distinguishing
  dependency, compatibility, corruption, resource, and trust failures. Never
  retry with a less restricted loader. Logs/metadata must not dump rows, weights,
  credentials, or signed URI parameters. Schemas/feature names can be sensitive;
  retain only required metadata under existing access controls.

### Frontend Considerations

Preserve Dataset/Model classification, artifact links, and lineage. Show bounded,
escaped type/format/codec metadata, retaining metadata-only views for legacy
artifacts. Update generated frontend types if IR changes. Rich dataset/model
previews are separate work; the UI/API must not reconstruct native models.

### KFP Local Considerations

`SubprocessRunner` and `DockerRunner` must use the same bindings, bundles, and
validation, including direct component calls and nested pipeline execution.
Preserve metadata through local inputs, outputs, and caches; do not shortcut
serialization by sharing host objects. Local `.output`/`.outputs` remain
artifact handles. Missing dependencies fail as they do remotely.

### Migration Strategy

Require explicit alpha opt-in. Existing parameters and artifact/path interfaces
are unchanged and remain the rollback path. Typed producers can feed
bundle-aware artifact consumers; typed consumers need valid contracts.
Document dependency/image setup, bundle layout, compatibility, and adoption of
legacy artifacts. No replacement store or database migration is expected.

### Test Plan

- [ ] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

These are implementation requirements, not completed production coverage.

#### Prerequisite testing updates

Add shared codec-conformance and previous-writer compatibility fixtures.
Measure current DSL/compiler/local-runner coverage before implementation;
baselines have not been collected for this draft.

#### Unit Tests

- Every supported type/format pair, dtype/state guarantees, empty values,
  incompatible versions, and model prediction equivalence after reconstruction.
- Tiny CPU transformer fixtures: offline logits/generation equivalence,
  sharded/tied weights, and rejection of missing shards or remote-code requests.
- Annotation/default resolution, named outputs, reader selection, compiled
  reloads, rejected root artifact inputs, and unchanged parameter/path behavior.
- Corrupt/oversized manifests, bounded compressed payloads, path confinement,
  forbidden loaders, redacted errors, and partial writes.
- Contract-aware remote/local cache keys, including custom keys.

#### E2E tests

- DataFrames and models across nested sub-pipelines, fan-out and artifact
  consumers; assert stable references, no extra DAG-boundary serialization, and
  mutation isolation.
- Conditions, optional artifacts, `ParallelFor`/`Collected`, retries, cache hits,
  missing bundles, and workspaces, with remote and both local runners.
  Alpha covers explicit list consumers; beta adds bounded typed-list loading.

#### Integration tests

- Generated IR, backend/launcher/executor propagation, unsupported-runtime
  rejection, imports/adoption, and local direct-task/pipeline parity.
- Optional-dependency isolation, generated component bindings, and `--no-deps`.
- Existing access restrictions on imported/cached/workspace bundles and UI
  metadata/lineage without native reconstruction.

Before beta, benchmark against manual IO using identical codecs/data/images;
agree on time/peak-memory budgets rather than assume a performance improvement.

### Graduation Criteria

- **Alpha:** Explicit opt-in, version/runtime checks, all three tabular libraries
  with both formats, a bounded sklearn subset, XGBoost's three classes with both
  formats, nested/local parity, artifact escape hatches, and adapter security
  review. No default pickle fallback.
- **Beta:** Documented custom adapters, supported NumPy/Keras/PyTorch and Hugging
  Face Transformers subsets, control-flow/list coverage, previous-writer
  compatibility, migration guidance, and measured resource budgets. Framework
  maturity may differ.
- **Stable:** Approved API/support ownership, dependency/format deprecation
  policy, upgrade/rollback coverage, and real-user feedback.

## Implementation History

- 2026-10-14: Drafted this proposal. A separate isolated OrbStack prototype
  demonstrated DataFrame formats and native XGBoost round trips through nested
  sub-pipelines. The prototype is not included in this KEP-only PR and does not
  implement production SDK/IR support or satisfy graduation requirements.
- 2026-10-15: Assigned KEP-14892 to match the
  [proposal PR](https://github.com/kubeflow/pipelines/pull/14892).

## Drawbacks

KFP takes responsibility for codec compatibility and a growing framework
matrix. Port contracts require coordinated SDK/IR/runtime changes; eager loading
can hide IO costs, and existing serving consumers may need bundle adapters.

## Alternatives

| Alternative | Tradeoff |
| --- | --- |
| Keep manual IO or helper libraries | No framework burden in KFP, but repeated boilerplate and weak reusable interfaces remain. |
| Universal pickle | Convenient, but unsafe as an implicit loader and dependency-sensitive. |
| One encoding for everything | Does not preserve every tabular type or reconstruct framework-specific models. |
| Shared-memory service first | Adds scheduling/lifecycle complexity without solving durable serialization. |
| Artifact metadata or executor source only | Cannot fully express independently loaded consumer and sub-DAG port contracts. |

## Open Questions

1. What annotation/registration API and IR capability/version mechanism should
   carry the contract?
2. Which exact class/dtype/version guarantees and adapter owners are supportable,
   and should all three tabular libraries default to Parquet?

## References

- [Kubeflow KEP process and template](https://github.com/kubeflow/community/tree/master/proposals)
  and [KFP proposal requirements](../README.md).
- [scikit-learn persistence](https://scikit-learn.org/stable/model_persistence.html)
  and [XGBoost model IO](https://xgboost.readthedocs.io/en/stable/tutorials/saving_model.html).
- [Keras model loading](https://keras.io/api/models/model_saving_apis/model_saving_and_loading/)
  and [PyTorch serialization](https://docs.pytorch.org/docs/stable/notes/serialization.html).
- [Hugging Face Transformers model persistence](https://huggingface.co/docs/transformers/main_classes/model)
  and [custom-model loading](https://huggingface.co/docs/transformers/custom_models).
