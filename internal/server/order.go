package server

import (
	"context"
	"errors"
	"time"

	"github.com/darkin-kat/order/internal/domain"
	"github.com/darkin-kat/order/internal/repository"
	ordersv1 "github.com/darkin-kat/store-api/gen/orders/v1"
	productsv1 "github.com/darkin-kat/store-api/gen/products/v1"
	usrv1 "github.com/darkin-kat/store-api/gen/users/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoOrder(order *domain.Order) *ordersv1.Order {
	var orderStatus ordersv1.OrderStatus
	switch order.Status {
	case domain.OrderStatusCreated:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_CREATED
	case domain.OrderStatusPaid:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_PAID
	case domain.OrderStatusShipped:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_SHIPPED
	case domain.OrderStatusDelivered:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_DELIVERED
	case domain.OrderStatusCancelled:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		orderStatus = ordersv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}

	items := make(map[string]*ordersv1.OrderItem, len(order.Items))
	for productID, item := range order.Items {
		items[productID] = &ordersv1.OrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		}
	}

	return &ordersv1.Order{
		Id:          order.ID,
		UserId:      order.UserID,
		UserName:    order.UserName,
		UserEmail:   order.UserEmail,
		Status:      orderStatus,
		TotalAmount: order.TotalAmount,
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
		Items:       items,
	}
}

func (s *Server) CreateOrder(ctx context.Context, req *ordersv1.CreateOrderRequest) (*ordersv1.CreateOrderResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return nil, status.Error(codes.NotFound, "cart is empty")
		}
		return nil, status.Error(codes.Internal, "failed to get cart")
	}

	if len(cart.Items) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "cart is empty")
	}

	userResp, err := s.userClient.GetUser(ctx, &usrv1.GetUserRequest{
		Identifier: &usrv1.GetUserRequest_Id{Id: req.GetUserId()},
	})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return nil, status.Error(codes.FailedPrecondition, "user does not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	items := make(map[string]domain.OrderItem, len(cart.Items))
	var total float64

	for productID, cartItem := range cart.Items {
		productResp, err := s.productsClient.GetProduct(ctx, &productsv1.GetProductRequest{Id: productID})
		if err != nil {
			st, _ := status.FromError(err)
			if st.Code() == codes.NotFound {
				return nil, status.Errorf(codes.FailedPrecondition, "product %s no longer available", productID)
			}
			return nil, status.Error(codes.Internal, "failed to get product")
		}

		price := productResp.GetProduct().GetPrice()
		lineTotal := price * float64(cartItem.Quantity)
		total += lineTotal

		items[productID] = domain.OrderItem{
			ProductID:   productID,
			ProductName: productResp.GetProduct().GetName(),
			Quantity:    cartItem.Quantity,
			Price:       price,
		}
	}

	order := &domain.Order{
		ID:          uuid.New().String(),
		UserID:      req.GetUserId(),
		UserName:    userResp.GetUser().GetFirstName() + " " + userResp.GetUser().GetLastName(),
		UserEmail:   userResp.GetUser().GetEmail(),
		Items:       items,
		Status:      domain.OrderStatusCreated,
		TotalAmount: total,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdOrder, err := s.orderRepo.Create(ctx, order)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create order")
	}
	// TODO : транзакции в MongoDB
	cart.Items = make(map[string]domain.CartItem) // Clear the cart
	cart.UpdatedAt = time.Now()
	if _, err := s.cartRepo.Save(ctx, &cart); err != nil {
		return nil, status.Error(codes.Internal, "failed to clear cart after order creation")
	}

	return &ordersv1.CreateOrderResponse{
		Order: toProtoOrder(&createdOrder),
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *ordersv1.GetOrderRequest) (*ordersv1.GetOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order ID is required")
	}

	order, err := s.orderRepo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	return &ordersv1.GetOrderResponse{
		Order: toProtoOrder(&order),
	}, nil
}

func (s *Server) ListUserOrders(ctx context.Context, req *ordersv1.ListUserOrdersRequest) (*ordersv1.ListUserOrdersResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if req.GetOffset() <= 0 {
		req.Offset = 0 // Default offset
	}
	if req.GetLimit() <= 0 {
		req.Limit = 10 // Default limit
	}

	orders, err := s.orderRepo.ListUserOrders(ctx, req.GetUserId(), uint(req.GetLimit()), uint(req.GetOffset()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list user orders")
	}

	total, err := s.orderRepo.TotalUserOrders(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total user orders")
	}

	protoOrders := make([]*ordersv1.Order, len(orders))
	for i := range orders {
		protoOrders[i] = toProtoOrder(&orders[i])
	}

	return &ordersv1.ListUserOrdersResponse{
		Orders: protoOrders,
		Total:  int32(total),
	}, nil
}

func (s *Server) SetOrderStatus(ctx context.Context, req *ordersv1.SetOrderStatusRequest) (*ordersv1.SetOrderStatusResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order ID is required")
	}

	newStatus := toDomainOrderStatus(req.GetStatus())
	if newStatus == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid order status")
	}

	order, err := s.orderRepo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	if !isValidTransition(order.Status, newStatus) {
		return nil, status.Errorf(codes.FailedPrecondition, "invalid status transition from %s to %s", order.Status, newStatus)
	}

	updatedOrder, err := s.orderRepo.SetStatus(ctx, req.GetId(), newStatus)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update order status")
	}

	return &ordersv1.SetOrderStatusResponse{
		Order: toProtoOrder(&updatedOrder),
	}, nil
}

var allowedTransitions = map[domain.OrderStatus][]domain.OrderStatus{
	domain.OrderStatusCreated: {domain.OrderStatusPaid, domain.OrderStatusCancelled},
	domain.OrderStatusPaid:    {domain.OrderStatusShipped, domain.OrderStatusCancelled},
	domain.OrderStatusShipped: {domain.OrderStatusDelivered},
}

func isValidTransition(from, to domain.OrderStatus) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func toDomainOrderStatus(s ordersv1.OrderStatus) domain.OrderStatus {
	switch s {
	case ordersv1.OrderStatus_ORDER_STATUS_CREATED:
		return domain.OrderStatusCreated
	case ordersv1.OrderStatus_ORDER_STATUS_PAID:
		return domain.OrderStatusPaid
	case ordersv1.OrderStatus_ORDER_STATUS_SHIPPED:
		return domain.OrderStatusShipped
	case ordersv1.OrderStatus_ORDER_STATUS_DELIVERED:
		return domain.OrderStatusDelivered
	case ordersv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.OrderStatusCancelled
	default:
		return ""
	}
}
