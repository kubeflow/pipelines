.PHONY: all
all: go_pipelinespec go_cachekey

.PHONY: clean
clean:
	rm -rf v2alpha1/go

go_pipelinespec: v2alpha1/pipeline_spec.proto
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	mkdir -p v2alpha1/go/pipelinespec
	cd v2alpha1 && protoc -I=. \
		--go_out=go/pipelinespec \
		--go_opt=paths=source_relative \
		pipeline_spec.proto

go_cachekey: v2alpha1/pipeline_spec.proto v2alpha1/cache_key.proto
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	mkdir -p v2alpha1/go/cachekey
	cd v2alpha1 && protoc -I=. \
		--go_out=go/cachekey \
		--go_opt=paths=source_relative \
		cache_key.proto
