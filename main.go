// consignment-service/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	// Import the generated protobuf code
	pb "github.com/fusidic/consignment-service/proto/consignment"
	userService "github.com/fusidic/user-service/proto/user"
	vesselProto "github.com/fusidic/vessel-service/proto/vessel"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
)

const (
	defaultHost = "mongodb://localhost:27017"
)

func main() {
	// 创建一个新的服务 包括了一些 micro options
	srv := micro.NewService(

		// Name 必须要和 consignment.proto 中的包名保持一致
		micro.Name("consignment"),
		micro.Version("latest"),
		// 认证服务的中间件
		micro.WrapHandler(AuthWrapper),
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

// AuthWrapper 是一个高阶函数，传入一个 HandlerFunc 并返回一个函数。
// 返回的函数以 context，request，response 作为接口参数
// token 从 consignment-cli 的上下文中获取，并被传到 user-service 中进行认证
// 认证通过，返回的函数继续执行；认证不通过，报错并返回
func AuthWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		// 通过环境变量 DISABLE_AUTH 确认是否需要跳过认证服务
		if os.Getenv("DISABLE_AUTH") == "true" {
			return fn(ctx, req, rsp)
		}
		meta, ok := metadata.FromContext(ctx)
		if !ok {
			return errors.New("no auth metadata found in request")
		}

		// 注意这里是用的大写 (我也不是很清楚为什么要这样写)
		token := meta["Token"]
		log.Println("Authenticating with token: ", token)

		// 认证
		authClient := userService.NewUserServiceClient("user", client.DefaultClient)
		authResp, err := authClient.ValidateToken(context.Background(), &userService.Token{
			Token: token,
		})
		log.Println("Auth Resp: ", authResp)
		if err != nil {
			return err
		}
		err = fn(ctx, req, rsp)
		return err
	}
}
