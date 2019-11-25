build: clean
	env CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/ilert-cache cmd/main.go

clean:
	rm -rf ./bin

run:
	go run cmd/main.go