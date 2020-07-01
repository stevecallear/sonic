.PHONY: generate
generate:
	go get github.com/golang/mock/mockgen@latest 
	go generate ./...

.PHONY: test
test: generate
	go test -v

.PHONY: cover
cover: generate
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	rm coverage.out

.PHONY: build
build: test
	env GOOS=linux GOARCH=amd64 go build -o ./bin/sonic ./cmd/sonic
	env GOOS=windows GOARCH=amd64 go build -o ./bin/sonic.exe ./cmd/sonic