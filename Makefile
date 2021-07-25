.PHONY: generate
generate:
	go get github.com/golang/mock/mockgen@latest 
	go generate ./...

.PHONY: test
test: generate
	go test -v

.PHONY: cover
cover: generate
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	rm coverage.out