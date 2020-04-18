// consignment-service/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	// Import the generated protobuf code
	pb "github.com/fusidic/consignment-service/proto/consignment"
	vesselProto "github.com/fusidic/vessel-service/proto/vessel"
	micro "github.com/micro/go-micro"
)

const (
	defaultHost = "datastore:27017"
)

func main() {
	// 启动微服务consignment实例
	srv := micro.NewService(
		micro.Name("consignment"),
	)

	srv.Init()

	// 获取环境变量 DB_HOST
	uri := os.Getenv("DB_HOST")
	if uri == "" {
		uri = defaultHost
	}

	// 创建数据库连接
	client, err := CreateClient(context.Background(), uri, 0)
	if err != nil {
		log.Panic(err)
	}
	defer client.Disconnect(context.Background())

	// 从shippy数据库中读取 collection consignments
	consignmentCollection := client.Database("shippy").Collection("consignments")

	repository := &MongoRepository{consignmentCollection}
	vesselClient := vesselProto.NewVesselServiceClient("vessel", srv.Client())
	h := &handler{repository, vesselClient}

	// 注册handlers
	pb.RegisterShippingServiceHandler(srv.Server(), h)

	// 运行服务器
	if err := srv.Run(); err != nil {
		fmt.Println(err)
	}
}
