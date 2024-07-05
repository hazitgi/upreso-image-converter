dev: 
	go run ./cmd/main.go

build: 
	go build -o ./bin/url_shortner ./cmd/main.go

fmt:
	go fmt ./...
