protoc --go-grpc_out="require_unimplemented_servers=false:." --go_out="." proto/*.proto
rm -rf ../agent/proto
cp -r proto ../agent