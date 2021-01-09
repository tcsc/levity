.PHONY: deps gen-api tests tests-race test-binaries binaries system-tests cert user-cert user-cert-wrong-ca

ROOT=$(shell pwd)
GOBIN="$(ROOT)/bin"
GRPC_GO_GEN=$(GOBIN)/protoc-gen-go

TEST_PACKAGES=$(shell go list ./... | grep -v cmd_test)
TEST_COMMANDS=$(shell go list ./... | grep cmd_test/)
COMMANDS=$(shell go list ./... | grep cmd/)

OPENSSL?=/usr/local/bin/openssl
CERT_DIR=$(ROOT)/cert
SVR_CA_KEY="$(CERT_DIR)/svr-ca-key.pem"
SVR_CA_CERT="$(CERT_DIR)/svr-ca-cert.pem"
CLIENT_CA_KEY="$(CERT_DIR)/client-ca-key.pem"
CLIENT_CA_CERT="$(CERT_DIR)/client-ca-cert.pem"
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
	git clean -dxf $(GOBIN)
	go clean ./...

certs:
	@echo "Generating Root CA for signing server certificate"
	@bin/mk-ca -o $(CERT_DIR) --pfx svr --ou levity --cn ServerCA --openssl $(OPENSSL)

	@echo "Generating server key"
	@$(OPENSSL) genpkey -algorithm ed25519 -outform PEM -out $(SVR_KEY)

	@echo "Generating server signing request"
	@$(OPENSSL) req -new -key $(SVR_KEY) -out $(SVR_CSR) -config "$(CERT_DIR)/svr.conf"

	@echo "Countersigning server certificate"
	@$(OPENSSL) x509 -req -in $(SVR_CSR) -CA $(SVR_CA_CERT) -CAkey $(SVR_CA_KEY) -CAcreateserial -out $(SVR_CERT) -extfile $(CERT_DIR)/svr.conf  -extensions req_ext

	@echo "Creating Root CA for signing client certificates"
	@bin/mk-ca -o $(CERT_DIR) --pfx client --ou levity --cn ClientCA --openssl $(OPENSSL)

	@echo "Creating Alice's (valid) certificate"
	./bin/mk-user-cert -u alice -o ./cert --ca-cert $(CLIENT_CA_CERT) --ca-key $(CLIENT_CA_KEY) --openssl $(OPENSSL)

	@echo "Creating Chuck's (invalid) certificate"
	./bin/mk-user-cert -u chuck -o ./cert --ca-cert $(SVR_CA_CERT) --ca-key $(SVR_CA_KEY) --openssl $(OPENSSL)

user-cert: LOGIN ?= usr
user-cert:
	./bin/mk-user-cert -u $(LOGIN) -o ./cert --ca-cert $(CLIENT_CA_CERT) --ca-key $(CLIENT_CA_KEY) --openssl $(OPENSSL)

user-cert-wrong-ca: LOGIN ?= chuck
user-cert-wrong-ca:
	./bin/mk-user-cert -u $(LOGIN) -o ./cert --ca-cert $(SVR_CA_CERT) --ca-key $(SVR_CA_KEY) --openssl $(OPENSSL)