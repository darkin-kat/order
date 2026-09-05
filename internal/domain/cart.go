package domain

import "time"

type Cart struct {
	ID        string              `bson:"_id"`
	UserID    string              `bson:"user_id"`
	Items     map[string]CartItem `bson:"items"`
	UpdatedAt time.Time           `bson:"updated_at"`
}

type CartItem struct {
	ProductID string `bson:"product_id"`
	Quantity  int    `bson:"quantity"`
}
