set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o build/order-service ./services/order-service/cmd/main.go