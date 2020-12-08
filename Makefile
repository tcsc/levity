.PHONY: deps gen-api tests tests-race test-binaries

export GOBIN=$(shell pwd)/bin
export PATH := $(GOBIN):$(PATH)
GRPC_GO_GEN=$(GOBIN)/protoc-gen-go

TEST_PACKAGES=$(shell go list ./... | grep -v test_cmd)
TEST_COMMANDS=$(shell go list ./... | grep test_cmd)

deps:
	go mod download

gen-api:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		--go_opt=paths=source_relative \
		--experimental_allow_proto3_optional \
		api/levity.proto

# Binaries with well-defined behaviour that we can use for testing
test-binaries: 
	go build -o $(GOBIN) $(TEST_COMMANDS)

# run unit tests
tests: test-binaries
	@go test -v -cover $(TEST_PACKAGES)

# run unit tests with the race checker enabled
tests-race: test-binaries
	@go test -race $(TEST_PACKAGES)

clean:
	rm -f $(GOBIN)/*