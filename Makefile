default: lint test

lint:
	golangci-lint run --timeout=5m

publish:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh

test:
	go test -cover -v ./...
