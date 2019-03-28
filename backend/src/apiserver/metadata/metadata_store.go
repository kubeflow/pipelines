package metadata

import (
	"encoding/json"
	"ml_metadata/metadata_store/mlmetadata"
	mlpb "ml_metadata/proto/metadata_store_go_proto"
	"strings"

	argoWorkflow "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Store encapsulates a ML Metadata store.
type Store struct {
	mlmdStore *mlmetadata.Store
}

// NewStore creates a new Store, using mlmdStore as the backing ML Metadata
// store.
func NewStore(mlmdStore *mlmetadata.Store) *Store {
	return &Store{mlmdStore: mlmdStore}
}

// RecordOutputArtifacts records metadata on artifacts as parsed from the Argo
// output parameters in currentManifest. storedManifest represents the currently
// stored manifest for the run with id runID, and is used to ensure metadata is
// recorded at most once per artifact.
func (s *Store) RecordOutputArtifacts(runID, storedManifest, currentManifest string) error {
	storedWorkflow := &argoWorkflow.Workflow{}
	if err := json.Unmarshal([]byte(storedManifest), storedWorkflow); err != nil {
		return util.NewInternalServerError(err, "unmarshaling workflow failed")
	}

	currentWorkflow := &argoWorkflow.Workflow{}
	if err := json.Unmarshal([]byte(currentManifest), currentWorkflow); err != nil {
		return util.NewInternalServerError(err, "unmarshaling workflow failed")
	}

	completed := make(map[string]bool)
	for _, n := range storedWorkflow.Status.Nodes {
		if n.Completed() {
			completed[n.ID] = true
		}
	}

	for _, n := range currentWorkflow.Status.Nodes {
		if n.Completed() && !completed[n.ID] {
			// Newly completed node. Record output ml-metadata artifacts.
			if n.Outputs != nil {
				for _, output := range n.Outputs.Parameters {
					if output.ValueFrom == nil || output.Value == nil || !strings.HasPrefix(output.ValueFrom.Path, "/output/ml_metadata/") {
						continue
					}

					artifacts, err := parseTFXMetadata(*output.Value)
					if err != nil {
						return util.NewInvalidInputError("metadata parsing failure: %v", err)
					}

					if err := s.storeArtifacts(artifacts); err != nil {
						return util.NewInvalidInputError("artifact storing failure: %v", err)
					}
				}
			}
		}
	}

	return nil
}

func (s *Store) storeArtifacts(artifacts artifactStructs) error {
	for _, a := range artifacts {
		id, err := s.mlmdStore.PutArtifactType(
			a.ArtifactType, &mlmetadata.PutTypeOptions{AllFieldsMustMatch: true})
		if err != nil {
			return util.NewInternalServerError(err, "failed to register artifact type")
		}
		a.Artifact.TypeId = proto.Int64(int64(id))
		_, err = s.mlmdStore.PutArtifacts([]*mlpb.Artifact{a.Artifact})
		if err != nil {
			return util.NewInternalServerError(err, "failed to record artifact")
		}
	}
	return nil
}

type artifactStruct struct {
	ArtifactType *mlpb.ArtifactType `json:"artifact_type"`
	Artifact     *mlpb.Artifact     `json:"artifact"`
}

func (a *artifactStruct) UnmarshalJSON(b []byte) error {
	errorF := func(err error) error {
		return util.NewInvalidInputError("JSON Unmarshal failure: %v", err)
	}

	jsonMap := make(map[string]*json.RawMessage)
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return errorF(err)
	}

	a.ArtifactType = &mlpb.ArtifactType{}
	a.Artifact = &mlpb.Artifact{}

	if err := jsonpb.UnmarshalString(string(*jsonMap["artifact_type"]), a.ArtifactType); err != nil {
		return errorF(err)
	}

	if err := jsonpb.UnmarshalString(string(*jsonMap["artifact"]), a.Artifact); err != nil {
		return errorF(err)
	}

	return nil
}

type artifactStructs []*artifactStruct

func parseTFXMetadata(value string) (artifactStructs, error) {
	var tfxArtifacts artifactStructs

	if err := json.Unmarshal([]byte(value), &tfxArtifacts); err != nil {
		return nil, util.NewInternalServerError(err, "parse TFX metadata failure")
	}
	return tfxArtifacts, nil
}
