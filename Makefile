default:
	go build ./...

test:
	go test ./...

build:
	go build -o ./bin/gigbee2mqtt ./cmd/gigbee2mqtt/main.go

rpi_build:
	env GOOS=linux GOARCH=arm GOARM=6 go build -o ./bin/gigbee2mqtt ./cmd/gigbee2mqtt/main.go