.PHONY: test verify cross-build release

COMMAND := agentsync

test:
	go test -race -timeout 2m ./...

verify:
	test -z "$$(gofmt -l cmd internal test version.go)"
	go vet ./...
	go test -race -timeout 2m ./...
	$(MAKE) cross-build

cross-build:
	GOOS=darwin GOARCH=amd64 go build -o /tmp/$(COMMAND)-darwin-amd64 ./cmd/$(COMMAND)
	GOOS=darwin GOARCH=arm64 go build -o /tmp/$(COMMAND)-darwin-arm64 ./cmd/$(COMMAND)
	GOOS=linux GOARCH=amd64 go build -o /tmp/$(COMMAND)-linux-amd64 ./cmd/$(COMMAND)
	GOOS=linux GOARCH=arm64 go build -o /tmp/$(COMMAND)-linux-arm64 ./cmd/$(COMMAND)

release:
	sh scripts/build-release.sh dist
	sh scripts/verify-release.sh dist
