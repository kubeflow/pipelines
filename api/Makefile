PROTO_PATH=/usr/local/include/

gen_proto: v2alpha1/pipeline_spec.proto
	cd v2alpha1 && protoc -I=$(PROTO_PATH) -I=. \
		--go_out=go \
		--go_opt=paths=source_relative \
		pipeline_spec.proto
