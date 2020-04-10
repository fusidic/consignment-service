// shippy-service-consignment/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	// Import the generated protobuf code

	pb "github.com/fusidic/consignment-service/proto/consignment"
	vesselProto "github.com/fusidic/vessel-service/proto/vessel"
	micro "github.com/micro/go-micro"
)

type repository interface {
	Create(*pb.Consignment) (*pb.Consignment, error)
	GetAll() []*pb.Consignment
}

// Repository - 暂时假装一个数据库
type Repository struct {
	// 由于多个服务访问存在竞争，需要引入读写锁
	mu           sync.RWMutex
	consignments []*pb.Consignment
}

// Create 创建一个新的consignment
func (repo *Repository) Create(consignment *pb.Consignment) (*pb.Consignment, error) {
	repo.mu.Lock()
	updated := append(repo.consignments, consignment)
	repo.consignments = updated
	repo.mu.Unlock()
	return consignment, nil
}

// GetAll consignments
func (repo *Repository) GetAll() []*pb.Consignment {
	return repo.consignments
}

// Service 需要实现protobuf.service中的所有定义，可以直接在pb.go中查找需要实现
// 的方法以及函数签名。
type service struct {
	repo repository

	// 引入vessel-service的接口
	vesselClient vesselProto.VesselServiceClient
}

// CreateConsignment - 在proto中，我们给这个微服务定了两个方法，其中之
// 一就是这个CreateConsignment方法，它接受一个context以及proto中定义
// 的Consignment消息，这个Consignment是由gRPC的服务器处理后提供给你的
func (s *service) CreateConsignment(ctx context.Context, req *pb.Consignment, res *pb.Response) error {

	// 这里调用vessel-service中的方法，寻找合适的vessel
	vesselResponse, err := s.vesselClient.FindAvailable(context.Background(), &vesselProto.Specification{
		MaxWeight: req.Weight,
		Capacity:  int32(len(req.Containers)),
	})
	log.Printf("Found vessel: %s \n", vesselResponse.Vessel.Name)
	if err != nil {
		return err
	}

	// 从返回的vesselResponse中获取vessel id
	req.VesselId = vesselResponse.Vessel.Id

	// Save our consignment
	consignment, err := s.repo.Create(req)
	if err != nil {
		return err
	}

	// Return matching the `Response` message we created in our
	// protobuf definition.
	res.Created = true
	res.Consignment = consignment
	return nil
}

// GetConsignments 实现proto中定义的另一个方法
func (s *service) GetConsignments(ctx context.Context, req *pb.GetRequest, res *pb.Response) error {
	consignments := s.repo.GetAll()
	res.Consignments = consignments
	return nil
}

func main() {

	repo := &Repository{}

	// 注意我们这里使用micro.NewService()创建服务，而不是之前的grpc.NewServer()
	srv := micro.NewService(

		// This name must match the package name given in your protobuf definition
		// 注意：服务名必须和你在proto文件中定义的package名字相同
		micro.Name("consignment"),
		micro.Version("latest"),
	)

	// Init will parse the command line flags.
	srv.Init()

	vesselClient := vesselProto.NewVesselServiceClient("vessel", srv.Client())

	// Register handler
	pb.RegisterShippingServiceHandler(srv.Server(), &service{repo, vesselClient})

	// Run the server
	if err := srv.Run(); err != nil {
		fmt.Println(err)
	}
}
