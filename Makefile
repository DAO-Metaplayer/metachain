.PHONY: download-submodules
download-submodules: check-git
	git submodule init
	git submodule update

.PHONY: check-git
check-git:
	@which git > /dev/null || (echo "git is not installed. Please install and try again."; exit 1)

.PHONY: check-go
check-go:
	@which go > /dev/null || (echo "Go is not installed.. Please install and try again."; exit 1)

.PHONY: check-protoc
check-protoc:
	@which protoc > /dev/null || (echo "protoc is not installed. Please install and try again."; exit 1)

.PHONY: check-lint
check-lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint is not installed. Please install and try again."; exit 1)

.PHONY: check-npm
check-npm:
	@which npm > /dev/null || (echo "npm is not installed. Please install and try again."; exit 1)

.PHONY: bindata
bindata: check-go
	go-bindata -pkg chain -o ./chain/chain_bindata.go ./chain/chains

.PHONY: protoc
protoc: check-protoc
	protoc --go_out=. --go-grpc_out=. -I . -I=./validate --validate_out="lang=go:." \
	 ./server/proto/*.proto \
	 ./network/proto/*.proto \
	 ./txpool/proto/*.proto	\
	 ./consensus/metabft/**/*.proto \
	 ./consensus/mbft/**/*.proto

.PHONY: build
build: check-go check-git
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval VERSION = $(shell git tag --points-at $COMMIT_HASH))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval TIME = $(shell date))
	go build -o metad -ldflags="\
    	-X 'github.com/DAO-Metaplayer/metachain/versioning.Version=$(VERSION)' \
		-X 'github.com/DAO-Metaplayer/metachain/versioning.Commit=$(COMMIT_HASH)'\
		-X 'github.com/DAO-Metaplayer/metachain/versioning.Branch=$(BRANCH)'\
		-X 'github.com/DAO-Metaplayer/metachain/versioning.BuildTime=$(TIME)'" \
	main.go

.PHONY: lint
lint: check-lint
	golangci-lint run --config .golangci.yml

.PHONY: generate-bsd-licenses
generate-bsd-licenses: check-git
	./generate_dependency_licenses.sh BSD-3-Clause,BSD-2-Clause > ./licenses/bsd_licenses.json

.PHONY: test
test: check-go
	go test -race -shuffle=on -coverprofile coverage.out -timeout 20m `go list ./... | grep -v e2e`

.PHONY: fuzz-test
fuzz-test: check-go
	./scripts/fuzzAll

.PHONY: test-e2e
test-e2e: check-go
	go build -race -o artifacts/metachain .
	env CHAIN_BINARY=${PWD}/artifacts/metachain go test -v -timeout=30m ./e2e/...

.PHONY: test-e2e-mbft
test-e2e-mbft: check-go
	go build -o artifacts/metachain .
	env CHAIN_BINARY=${PWD}/artifacts/metachain E2E_TESTS=true E2E_LOGS=true \
	go test -v -timeout=1h30m ./e2e-mbft/e2e/...

.PHONY: test-property-mbft
test-property-mbft: check-go
	go build -o artifacts/metachain .
	env CHAIN_BINARY=${PWD}/artifacts/metachain E2E_TESTS=true E2E_LOGS=true go test -v -timeout=30m ./e2e-mbft/property/... \
	-rapid.checks=10

.PHONY: compile-core-contracts
compile-core-contracts: check-npm
	cd metacore-contracts && npm install && npm run compile
	$(MAKE) generate-smart-contract-bindings

.PHONY: generate-smart-contract-bindings
generate-smart-contract-bindings: check-go
	go run ./consensus/mbft/contractsapi/artifacts-gen/main.go
	go run ./consensus/mbft/contractsapi/bindings-gen/main.go

.PHONY: run-docker
run-docker:
	./scripts/cluster mbft --docker

.PHONY: stop-docker
stop-docker:
	./scripts/cluster mbft --docker stop

.PHONY: destroy-docker
destroy-docker:
	./scripts/cluster mbft --docker destroy

.PHONY: help
help:
	@echo "Available targets:"
	@printf "  %-35s - %s\n" "download-submodules" "Initialize and update Git submodules"
	@printf "  %-35s - %s\n" "bindata" "Generate Go binary data for chain"
	@printf "  %-35s - %s\n" "protoc" "Compile Protocol Buffers files"
	@printf "  %-35s - %s\n" "build" "Build the project"
	@printf "  %-35s - %s\n" "lint" "Run linters on the codebase"
	@printf "  %-35s - %s\n" "generate-bsd-licenses" "Generate BSD licenses"
	@printf "  %-35s - %s\n" "test" "Run unit tests"
	@printf "  %-35s - %s\n" "fuzz-test" "Run fuzz tests"
	@printf "  %-35s - %s\n" "test-e2e" "Run end-to-end tests"
	@printf "  %-35s - %s\n" "test-e2e-mbft" "Run end-to-end tests for MBFT"
	@printf "  %-35s - %s\n" "test-property-mbft" "Run property tests for MBFT"
	@printf "  %-35s - %s\n" "compile-core-contracts" "Compile metacore contracts"
	@printf "  %-35s - %s\n" "generate-smart-contract-bindings" "Generate smart contract bindings"
	@printf "  %-35s - %s\n" "run-docker" "Run Docker cluster for MBFT"
	@printf "  %-35s - %s\n" "stop-docker" "Stop Docker cluster for MBFT"
	@printf "  %-35s - %s\n" "destroy-docker" "Destroy Docker cluster for MBFT"
