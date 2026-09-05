package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/darkin-kat/order/internal/domain"
	"github.com/darkin-kat/order/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) Save(ctx context.Context, cart *domain.Cart) (domain.Cart, error) {
	filter := bson.M{"_id": cart.ID}
	update := bson.M{
		"$set": bson.M{
			"_id":        cart.ID,
			"user_id":    cart.UserID,
			"items":      cart.Items,
			"updated_at": cart.UpdatedAt,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var updatedCart domain.Cart
	err := r.Carts.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedCart)
	if err != nil {
		return domain.Cart{}, err
	}
	return updatedCart, nil
}

func (r *Repository) RemoveFromCart(ctx context.Context, userID string, productID string) (domain.Cart, error) {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$unset": bson.M{
			"items." + productID: "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedCart domain.Cart
	err := r.Carts.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedCart)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Cart{}, repository.ErrCartNotFound
		}
		return domain.Cart{}, err
	}
	return updatedCart, nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID string) (domain.Cart, error) {
	filter := bson.M{"user_id": userID}
	opts := options.FindOne()
	var cart domain.Cart
	err := r.Carts.FindOne(ctx, filter, opts).Decode(&cart)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Cart{}, repository.ErrCartNotFound
		}
		return domain.Cart{}, err
	}
	return cart, nil
}
