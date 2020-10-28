all: build

.PHONY: license
license:
	go run github.com/google/addlicense -c "Evan Cordell" -l apache -f LICENSE_HEADER ./cmd/**/*.go ./pkg/**/*.go

.PHONY: check
check:
	go run github.com/google/addlicense -c "Evan Cordell" -l apache -f LICENSE_HEADER -check  ./cmd/**/*.go ./pkg/**/*.go

build: check
	go build -mod=vendor ./cmd/...

kubectl-plugin:
	go build -mod=vendor -o kubectl-cue ./cmd/cuebectl
	mv ./kubectl-cue /usr/local/bin/kubectl-cue

