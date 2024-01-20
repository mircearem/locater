build-arm:
	@env GOOS=linux GOARCH=arm GOARM=7 go build -o ./bin/geo-gps
	@chmod +x ./bin/geo-gps

build:
	go build -o ./bin/geo-gps

run: build
	@./bin/geo-gps

test:
	@go test -v ./... 