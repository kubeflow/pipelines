MLMD_VERSION=v0.26.0
PROTO_DIR=ml_metadata/proto
OUT_DIR=go_client

all: proto_gen

.PHONY: update
update:
	# Please first edit MLMD_VERSION in Makefile
	$(MAKE) clean all

.PHONY: clean
clean:
	rm -rf proto
	rm -rf $(OUT_DIR)

.PHONY: proto_gen
proto_gen: $(PROTO_DIR)/metadata_store.proto $(PROTO_DIR)/metadata_store_service.proto
	\
	mkdir -p $(OUT_DIR) && \
	protoc -I=. --go_out=$(OUT_DIR) \
		--go-grpc_out=$(OUT_DIR) \
		$(PROTO_DIR)/metadata_store.proto $(PROTO_DIR)/metadata_store_service.proto \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative

$(PROTO_DIR)/metadata_store.proto:
	mkdir -p ml_metadata/proto
	cd ml_metadata/proto && curl -LO https://raw.githubusercontent.com/google/ml-metadata/$(MLMD_VERSION)/ml_metadata/proto/metadata_store.proto

$(PROTO_DIR)/metadata_store_service.proto:
	mkdir -p ml_metadata/proto
	cd ml_metaddata/proto && curl -LO https://raw.githubusercontent.com/google/ml-metadata/$(MLMD_VERSION)/ml_metadata/proto/metadata_store_service.proto
