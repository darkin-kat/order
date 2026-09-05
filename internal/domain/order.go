package domain

import "time"

type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID          string               `bson:"_id"`
	UserID      string               `bson:"user_id"`
	UserName    string               `bson:"user_name"`
	UserEmail   string               `bson:"user_email"`
	Items       map[string]OrderItem `bson:"items"`
	Status      OrderStatus          `bson:"status"`
	TotalAmount float64              `bson:"total_amount"`
	CreatedAt   time.Time            `bson:"created_at"`
	UpdatedAt   time.Time            `bson:"updated_at"`
}

type OrderItem struct {
	ProductID   string  `bson:"product_id"`
	ProductName string  `bson:"product_name"`
	Quantity    int     `bson:"quantity"`
	Price       float64 `bson:"price"`
}
