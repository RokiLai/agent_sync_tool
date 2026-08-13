.PHONY: test verify compatibility cross-build release

test:
	go test -race ./...

verify:
	gofmt -w cmd internal test
	go vet ./...
	go test -race ./...
	sh scripts/verify-shell-contracts.sh

compatibility:
	sh scripts/run-implementation-contracts.sh all

cross-build:
	GOOS=darwin GOARCH=amd64 go build -o /tmp/aic-darwin-amd64 ./cmd/aic
	GOOS=darwin GOARCH=arm64 go build -o /tmp/aic-darwin-arm64 ./cmd/aic
	GOOS=linux GOARCH=amd64 go build -o /tmp/aic-linux-amd64 ./cmd/aic
	GOOS=linux GOARCH=arm64 go build -o /tmp/aic-linux-arm64 ./cmd/aic

release:
	sh scripts/build-release.sh dist
	sh scripts/verify-release.sh dist
