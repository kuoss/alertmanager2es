PROJECT_NAME	:= $(shell basename $(CURDIR))
GIT_TAG			:= $(shell git describe --dirty --tags --always)
GIT_COMMIT		:= $(shell git rev-parse --short HEAD)
LDFLAGS			:= -X "main.gitTag=$(GIT_TAG)" -X "main.gitCommit=$(GIT_COMMIT)" -extldflags "-static" -s -w

FIRST_GOPATH	:= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOLANGCI_LINT   := $(shell pwd)/bin/golangci-lint
GOSEC_BIN		:= $(shell pwd)/bin/gosec

GOLANGCI_LINT_VERSION := v2.7.2

.PHONY: all
all: build

.PHONY: clean
clean:
	git clean -Xfd .

.PHONY: build
build:
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o $(PROJECT_NAME) .

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: image
image: build
	docker build -t $(PROJECT_NAME):$(GIT_TAG) .

build-push-development:
	docker buildx create --use
	docker buildx build -t kuoss/$(PROJECT_NAME):development --platform linux/amd64,linux/arm,linux/arm64 --push .

.PHONY: test
test:
	go test ./...

.PHONY: dependencies
dependencies:
	go mod vendor

.PHONY: check-release
check-release: vendor lint gosec test

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: gosec
gosec: $(GOSEC_BIN)
	$(GOSEC_BIN) ./...

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT):
	mkdir -p ./bin
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh \
		| sed -e '/install -d/d' \
		| sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)

.PHONY: compose
compose:
	cd development; docker compose down; docker compose up -d
