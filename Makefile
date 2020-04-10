build:
	protoc -I. --go_out=plugins=grpc:. \
	  proto/consignment/consignment.proto
	
	GOOS=linux GOARCH=amd64 go build
	docker build -t consignment-service .

run:
	