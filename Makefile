VERSION := $(shell git describe --always --tags HEAD)$(and $(shell git status --porcelain),+$(shell scripts/worktree-hash.sh))

all: build

.PHONY: license
license:
	go run github.com/google/addlicense -c "Evan Cordell" -l apache -f LICENSE_HEADER ./cmd/**/*.go ./pkg/**/*.go

.PHONY: check
check:
	go run github.com/google/addlicense -c "Evan Cordell" -l apache -f LICENSE_HEADER -check  ./cmd/**/*.go ./pkg/**/*.go

build: check
	go build -mod=vendor -ldflags '-X ./pkg/internal/version.Version=$(VERSION)' -o bin/cuebectl$(go env GOEXE) ./cmd/...

kubectl-plugin: check
	go build -mod=vendor -ldflags '-X ./pkg/internal/version.Version=$(VERSION)' -o bin/kubectl-cue ./cmd/cuebectl

.PHONY: install-kubectl
install-kubectl: kubectl-plugin
	mv ./bin/kubectl-cue /usr/local/bin/kubectl-cue

