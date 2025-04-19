proto:
	protoc \
		--proto_path=api \
		--go_out=./internal/pb \
		--go_opt=paths=source_relative \
		--go-grpc_out=./internal/pb \
		--go-grpc_opt=paths=source_relative \
		api/clerk.proto api/trace.proto

build:
	go build -o ./bin/hammer ./cmd/hammer
	go build -o ./bin/recorder ./cmd/recorder
	go build -o ./bin/searcher ./cmd/searcher
