HOSTNAME=registry.terraform.io
NAMESPACE=tsuru
NAME=rpaas
BINARY=terraform-provider-${NAME}
VERSION=$(shell git describe --tags $(git rev-list --tags --max-count=1) | tr -d v)
UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)
ifeq ($(UNAME_S),Linux)
	OS := linux
	UNAME_P := $(shell uname -m)
endif
ifeq ($(UNAME_S),Darwin)
	OS := darwin
	UNAME_P := $(shell uname -m)
endif

ifeq ($(UNAME_P),x86_64)
	ARCH := amd64
endif

ifneq ($(filter %86,$(UNAME_P)),)
	ARCH := 386
endif
ifneq ($(filter arm%,$(UNAME_P)),)
	ARCH := arm
endif
ifeq ($(UNAME_P),arm64)
	ARCH := arm64
endif

OS_ARCH=${OS}_${ARCH}

TFPLUGINDOCS_VERSION ?= v0.16.0

default: lint test build install

build:
	CGO_ENABLED=0 go build -ldflags="-X github.com/tsuru/terraform-provider-rpaas/internal/provider.Version=$(VERSION)" -o ${BINARY}

lint:
	golangci-lint run

release:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=freebsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_freebsd_386
	GOOS=freebsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_freebsd_amd64
	GOOS=freebsd GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_freebsd_arm
	GOOS=linux GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_linux_386
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=linux GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_linux_arm
	GOOS=openbsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_openbsd_386
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_openbsd_amd64
	GOOS=solaris GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_solaris_amd64
	GOOS=windows GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_windows_386
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	cp ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

uninstall:
	rm -Rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}

test:
	TF_ACC=1 TF_ACC_TERRAFORM_VERSION=1.4.4 go test ./... -v

debug_test:
	TF_LOG=debug make test

generate-docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION)
	go generate
