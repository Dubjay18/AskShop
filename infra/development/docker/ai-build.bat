set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o build/ai-service ./services/ai-service/cmd/main.go