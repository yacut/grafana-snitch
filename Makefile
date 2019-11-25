build: clean
	env CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/grafana-snitch ./

clean:
	rm -rf ./bin

run:
	go run ./