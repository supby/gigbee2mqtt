default:
	go build ./...

test:
	go test ./...

build:
	go build -o ./bin/gigbee2mqtt gigbee2mqtt.go

rpi_build:
	env GOOS=linux GOARCH=arm GOARM=6 go build -o ./bin/gigbee2mqtt gigbee2mqtt.go