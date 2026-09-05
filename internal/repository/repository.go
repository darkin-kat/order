package repository

import (
	"context"

	"errors"

	"github.com/darkin-kat/order/internal/domain"
)

var (
	ErrCartNotFound  = errors.New("cart not found")
	ErrOrderNotFound = errors.New("order not found")
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) (domain.Order, error)
	GetByID(ctx context.Context, id string) (domain.Order, error)
	SetStatus(ctx context.Context, id string, status domain.OrderStatus) (domain.Order, error)
	ListUserOrders(ctx context.Context, userID string, limit uint, offset uint) ([]domain.Order, error)
	TotalUserOrders(ctx context.Context, userID string) (int64, error)
}

type CartRepository interface {
	Save(ctx context.Context, cart *domain.Cart) (domain.Cart, error)
	GetByUserID(ctx context.Context, userID string) (domain.Cart, error)
	RemoveFromCart(ctx context.Context, userID string, productID string) (domain.Cart, error)
}
