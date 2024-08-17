default: arm

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm GOARM=7 go build -o prom-proxy ./main.go

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build -o prom-proxy ./main.go
