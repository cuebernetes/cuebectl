all: build

build:
	go build -mod=vendor ./cmd/...

kubectl-plugin:
	go build -mod=vendor -o kubectl-cue ./cmd/cuebectl

