default: lint test

lint:
	golangci-lint run --timeout=5m

publish:
	bash ./ci/build.sh

test:
	go test -cover -v ./...
