package server

import (
	"github.com/darkin-kat/order/internal/repository"
	ordersv1 "github.com/darkin-kat/store-api/gen/orders/v1"
	productsv1 "github.com/darkin-kat/store-api/gen/products/v1"
	usrv1 "github.com/darkin-kat/store-api/gen/users/v1"
)

type Server struct {
	ordersv1.UnimplementedOrderServiceServer
	ordersv1.UnimplementedCartServiceServer

	cartRepo  repository.CartRepository
	orderRepo repository.OrderRepository

	userClient     usrv1.UserServiceClient
	productsClient productsv1.ProductsServiceClient
}

func NewServer(
	cartRepo repository.CartRepository,
	orderRepo repository.OrderRepository,
	userClient usrv1.UserServiceClient,
	productsClient productsv1.ProductsServiceClient,
) *Server {
	return &Server{
		cartRepo:       cartRepo,
		orderRepo:      orderRepo,
		userClient:     userClient,
		productsClient: productsClient,
	}
}
