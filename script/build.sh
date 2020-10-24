# !/bin/sh
GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build -o ./release/nezha-$1 cmd/$1/main.go