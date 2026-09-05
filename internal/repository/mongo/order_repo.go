package mongo

import (
	"context"
	"errors"

	"github.com/darkin-kat/order/internal/domain"
	"github.com/darkin-kat/order/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) Create(ctx context.Context, order *domain.Order) (domain.Order, error) {
	_, err := r.Orders.InsertOne(ctx, order)
	if err != nil {
		return domain.Order{}, err
	}
	return *order, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	filter := bson.M{"_id": id}
	var order domain.Order
	err := r.Orders.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Order{}, repository.ErrOrderNotFound
		}
		return domain.Order{}, err
	}
	return order, nil
}

func (r *Repository) SetStatus(ctx context.Context, id string, status domain.OrderStatus) (domain.Order, error) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"status": status}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var order domain.Order
	err := r.Orders.FindOneAndUpdate(ctx, filter, update, opts).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Order{}, repository.ErrOrderNotFound
		}
		return domain.Order{}, err
	}
	return order, nil
}

func (r *Repository) ListUserOrders(ctx context.Context, userID string, limit uint, offset uint) ([]domain.Order, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.M{"created_at": -1})

	cursor, err := r.Orders.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []domain.Order
	for cursor.Next(ctx) {
		var order domain.Order
		if err = cursor.Decode(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err = cursor.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *Repository) TotalUserOrders(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{"user_id": userID}
	count, err := r.Orders.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}
