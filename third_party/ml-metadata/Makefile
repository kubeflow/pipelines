PROTO_DIR=ml_metadata/proto
OUT_DIR=tmp
GO_DIR=go/ml_metadata
GO_PACKAGE=github.com/kubeflow/pipelines/third_party/ml-metadata/$(GO_DIR)

all: proto_gen

.PHONY: update
update:
	# Please first edit MLMD version in the file ./VERSION
	$(MAKE) clean all

.PHONY: clean
clean:
	rm -rf ml_metadata/proto
	rm -rf $(GO_DIR)

.PHONY: proto_gen
proto_gen: $(PROTO_DIR)/metadata_store.proto $(PROTO_DIR)/metadata_store_service.proto
	mkdir -p $(OUT_DIR) && \
	protoc -I=. --go_out=$(OUT_DIR) \
		--go-grpc_out=$(OUT_DIR) \
		$(PROTO_DIR)/metadata_store.proto $(PROTO_DIR)/metadata_store_service.proto \
		--go_opt=paths=source_relative \
		--go_opt=M$(PROTO_DIR)/metadata_store.proto=$(GO_PACKAGE) \
		--go_opt=M$(PROTO_DIR)/metadata_store_service.proto=$(GO_PACKAGE) \
		--go-grpc_opt=paths=source_relative && \
	mkdir -p $(GO_DIR) && \
	mv $(OUT_DIR)/$(PROTO_DIR)/* $(GO_DIR)/ && \
	rm -rf $(OUT_DIR)

$(PROTO_DIR)/metadata_store.proto:
	mkdir -p ml_metadata/proto
	tag="v$$(cat VERSION)" && cd ml_metadata/proto && curl -LO "https://raw.githubusercontent.com/google/ml-metadata/$${tag}/ml_metadata/proto/metadata_store.proto"

$(PROTO_DIR)/metadata_store_service.proto:
	mkdir -p ml_metadata/proto
	tag="v$$(cat VERSION)" && cd ml_metadata/proto && curl -LO "https://raw.githubusercontent.com/google/ml-metadata/$${tag}/ml_metadata/proto/metadata_store_service.proto"
