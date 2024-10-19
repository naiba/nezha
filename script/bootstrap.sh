swag init -g cmd/dashboard/main.go -o cmd/dashboard/docs
protoc --go-grpc_out="require_unimplemented_servers=false:." --go_out="." proto/*.proto
rm -rf ../agent/proto
cp -r proto ../agent