package server

import (
	"context"
	"errors"
	"time"

	"github.com/darkin-kat/order/internal/domain"
	"github.com/darkin-kat/order/internal/repository"
	ordersv1 "github.com/darkin-kat/store-api/gen/orders/v1"
	productsv1 "github.com/darkin-kat/store-api/gen/products/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) AddItemsToCart(ctx context.Context, req *ordersv1.AddItemsToCartRequest) (*ordersv1.AddItemsToCartResponse, error) {
	if req.GetUserId() == "" || len(req.GetItems()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and items are required")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, req.GetUserId())
	if err != nil {
		if !errors.Is(err, repository.ErrCartNotFound) {
			return nil, status.Error(codes.Internal, "failed to get cart")
		}
		cart = domain.Cart{
			ID:     uuid.New().String(),
			UserID: req.GetUserId(),
			Items:  make(map[string]domain.CartItem),
		}
	}

	for _, item := range req.GetItems() {
		if item.GetProductId() == "" || item.GetQuantity() <= 0 {
			return nil, status.Error(codes.InvalidArgument, "product_id and positive quantity are required for each item")
		}

		if _, err = s.productsClient.GetProduct(ctx, &productsv1.GetProductRequest{Id: item.GetProductId()}); err != nil {
			st, _ := status.FromError(err)
			if st.Code() == codes.NotFound {
				return nil, status.Errorf(codes.InvalidArgument, "product_id %s not found", item.GetProductId())
			}
			return nil, status.Error(codes.Internal, "failed to validate product")
		}

		exist := cart.Items[item.GetProductId()]
		exist.ProductID = item.GetProductId()
		exist.Quantity += int(item.GetQuantity())
		cart.Items[item.GetProductId()] = exist
	}
	cart.UpdatedAt = time.Now()

	saved, err := s.cartRepo.Save(ctx, &cart)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to save cart")
	}

	protoCart, err := s.toProtoCart(ctx, &saved)
	if err != nil {
		return nil, err
	}

	return &ordersv1.AddItemsToCartResponse{
		Cart: protoCart,
	}, nil
}

func (s *Server) toProtoCart(ctx context.Context, cart *domain.Cart) (*ordersv1.Cart, error) {
	items := make(map[string]*ordersv1.CartItemView, len(cart.Items))
	var total float64

	for productID, item := range cart.Items {
		product, err := s.productsClient.GetProduct(ctx, &productsv1.GetProductRequest{Id: productID})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to load product for cart")
		}

		price := product.GetProduct().GetPrice() // если добавите Price в Product; пока условно
		lineTotal := price * float64(item.Quantity)
		total += lineTotal

		items[productID] = &ordersv1.CartItemView{
			ProductId:   productID,
			ProductName: product.GetProduct().GetName(),
			Price:       price,
			Quantity:    int32(item.Quantity),
			LineTotal:   lineTotal,
		}
	}

	return &ordersv1.Cart{
		Id:          cart.ID,
		UserId:      cart.UserID,
		Items:       items,
		TotalAmount: total,
		UpdatedAt:   timestamppb.New(cart.UpdatedAt),
	}, nil
}

func (s *Server) GetCart(ctx context.Context, req *ordersv1.GetCartRequest) (*ordersv1.GetCartResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return &ordersv1.GetCartResponse{
				Cart: &ordersv1.Cart{
					Id:          "",
					UserId:      req.GetUserId(),
					Items:       make(map[string]*ordersv1.CartItemView),
					TotalAmount: 0,
				},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get cart")
	}
	//cart = domain.Cart{
	//	ID:        uuid.New().String(),
	//	UserID:    req.GetUserId(),
	//	Items:     make(map[string]*domain.CartItem),
	//	UpdatedAt: time.Now(),
	//}
	//saved, saveErr := s.cartRepo.Save(ctx, &cart)
	//if saveErr != nil {
	//	return nil, status.Error(codes.Internal, "failed to create cart")
	//}
	//cart = saved

	protoCart, err := s.toProtoCart(ctx, &cart)
	if err != nil {
		return nil, err
	}

	return &ordersv1.GetCartResponse{
		Cart: protoCart,
	}, nil
}

func (s *Server) RemoveFromCart(ctx context.Context, req *ordersv1.RemoveFromCartRequest) (*ordersv1.RemoveFromCartResponse, error) {
	if req.GetUserId() == "" || req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and product_id are required")
	}

	cart, err := s.cartRepo.RemoveFromCart(ctx, req.GetUserId(), req.GetProductId())
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return nil, status.Error(codes.NotFound, "cart not found")
		}
		return nil, status.Error(codes.Internal, "failed to remove item from cart")
	}

	protoCart, err := s.toProtoCart(ctx, &cart)
	if err != nil {
		return nil, err
	}

	return &ordersv1.RemoveFromCartResponse{
		Cart: protoCart,
	}, nil
}
