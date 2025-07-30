set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o build/product-service ./services/product-service/cmd/main.go