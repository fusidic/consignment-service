// consignment-service/handler.go
package main

import (
	"context"
	"errors"

	pb "github.com/fusidic/consignment-service/proto/consignment"
	vesselProto "github.com/fusidic/vessel-service/proto/vessel"
)

type handler struct {
	repository
	vesselClient vesselProto.VesselServiceClient
}

// CreateConsignment - 只在服务中创建一个create方法，需要context, request, response作为参数
// 这些参数都由gRPC服务器进行处理
func (s *handler) CreateConsignment(ctx context.Context, req *pb.Consignment, res *pb.Response) error {

	// 这里我们调用vessel的服务实例，并且传入consignment的重量和container的数量
	vesselResponse, err := s.vesselClient.FindAvailable(ctx, &vesselProto.Specification{
		MaxWeight: req.Weight,
		Capacity:  int32(len(req.Containers)),
	})
	if vesselResponse == nil {
		return errors.New("error fetching vessel, returned nil")
	}

	if err != nil {
		return err
	}

	// 将req.VesselId赋值为vessel服务计算出来最适合的vessel
	req.VesselId = vesselResponse.Vessel.Id

	// 创建货运订单
	if err = s.repository.Create(ctx, MarshalConsignment(req)); err != nil {
		return err
	}

	res.Created = true
	res.Consignment = req
	return nil
}

// GetConsignments -
func (s *handler) GetConsignments(ctx context.Context, req *pb.GetRequest, res *pb.Response) error {
	consignments, err := s.repository.GetAll(ctx)
	if err != nil {
		return err
	}
	res.Consignments = UnmarshalConsignmentCollection(consignments)
	return nil
}
