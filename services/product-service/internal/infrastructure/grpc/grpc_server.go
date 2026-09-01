package grpc_server

import (
	"askshop/services/product-service/internal/domain"
	"askshop/shared/proto"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	addr string
	svc  domain.ProductService
}

type ProductServiceGRPCHandler struct {
	proto.UnimplementedProductServiceServer
	service domain.ProductService
}

// GetProduct fetches a product by ID (only ID is supported in this minimal handler).
func (h *ProductServiceGRPCHandler) GetProduct(ctx context.Context, req *proto.ProductRequest) (*proto.Product, error) {
	idReq, ok := req.Identifier.(*proto.ProductRequest_Id)
	if !ok {
		// Only ID lookup implemented in this minimal wiring
		return nil, status.Errorf(codes.Unimplemented, "lookup by non-ID not supported")
	}
	p, err := h.service.GetProductByID(ctx, idReq.Id)
	if err != nil {
		return nil, err
	}
	return toProtoProduct(p), nil
}

func toProtoProduct(p *domain.Product) *proto.Product {
	if p == nil {
		return nil
	}
	out := &proto.Product{
		Id:            p.ID.String(),
		Name:          p.Name,
		Slug:          p.Slug,
		Sku:           p.SKU,
		Description:   p.Description,
		Currency:      p.Currency,
		Status:        p.Status,
		Tags:          append([]string(nil), p.Tags...),
		PriceCents:    p.PriceCents,
		StockQuantity: int32(p.StockQuantity),
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
	// Images and Categories omitted for brevity
	return out
}

func NewGRPCServer(addr string, svc domain.ProductService) *GRPCServer {
	return &GRPCServer{addr: addr, svc: svc}
}

func (s *GRPCServer) Start() {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	proto.RegisterProductServiceServer(grpcServer, &ProductServiceGRPCHandler{service: s.svc})
	log.Println("gRPC server listening on", s.addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}

}
