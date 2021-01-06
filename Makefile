.PHONY: deps gen-api tests tests-race test-binaries binaries system-tests cert

ROOT=$(shell pwd)
GOBIN="$(ROOT)/bin"
GRPC_GO_GEN=$(GOBIN)/protoc-gen-go

TEST_PACKAGES=$(shell go list ./... | grep -v cmd_test)
TEST_COMMANDS=$(shell go list ./... | grep cmd_test/)
COMMANDS=$(shell go list ./... | grep cmd/)

OPENSSL?=/usr/local/bin/openssl
CERT_DIR=$(ROOT)/cert
CA_KEY="$(CERT_DIR)/ca-key.pem"
CA_CERT="$(CERT_DIR)/ca-cert.pem"
SVR_KEY="$(CERT_DIR)/svr-key.pem"
SVR_CERT="$(CERT_DIR)/svr-cert.pem"
SVR_CSR="$(CERT_DIR)/svr-req.csr"

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
tests: binaries test-binaries
	@PATH=$(GOBIN):$(PATH) go test -v -cover $(TEST_PACKAGES)

binaries:
	go build -o $(GOBIN) $(COMMANDS)

system-tests: binaries
	@PATH=$(GOBIN):$(PATH) go test -v ./cmd

# run unit tests with the race checker enabled
tests-race: test-binaries binaries
	@PATH=$(GOBIN):$(PATH) go test -race $(TEST_PACKAGES)

clean:
	rm -f $(GOBIN)/*
	go clean ./...

certs: $(CA_CERT) $(SVR_CERT)

$(CA_KEY):
	@echo "Generating CA Key"
	@$(OPENSSL) genpkey -algorithm ed25519 -outform PEM -out $(CA_KEY)

$(CA_CERT): $(CA_KEY)
	@echo "Generating CA Root Certificate"
	@$(OPENSSL) req -x509 -newkey rsa:4096 -days 365 -nodes -key $(CA_KEY) -out $(CA_CERT) -subj "/C=AU/ST=Victoria/L=Melbourne/OU=SelfSignedCA/CN=localhost/emailAddress=trent.clarke@gmail.com"

$(SVR_KEY):
	@echo "Generating Server Key"
	@$(OPENSSL) genpkey -algorithm ed25519 -outform PEM -out $(SVR_KEY)

$(SVR_CSR): $(SVR_KEY)
	@echo "Generating Server Certificate Signing Request"
	@$(OPENSSL) req -new -key $(SVR_KEY) -out $(SVR_CSR) -config "$(CERT_DIR)/svr.conf"

$(SVR_CERT): $(CA_CERT) $(SVR_CSR)
	@echo "Signing Server Certificate"
	@$(OPENSSL) x509 -req -in $(SVR_CSR) -CA $(CA_CERT) -CAkey $(CA_KEY) -CAcreateserial -out $(SVR_CERT) -extfile $(CERT_DIR)/svr.conf  -extensions req_ext