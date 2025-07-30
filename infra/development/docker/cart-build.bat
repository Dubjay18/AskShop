set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o build/cart-service ./services/cart-service/cmd/main.go