PRODUCT_PROTO_DIR=shared/proto
PRODUCT_PROTO_FILES=$(shell find $(PRODUCT_PROTO_DIR) -name "*.proto")

product-proto-gen:
	protoc -I . \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  $(PRODUCT_PROTO_FILES)