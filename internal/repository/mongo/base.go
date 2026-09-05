package mongo

import "go.mongodb.org/mongo-driver/mongo"

type Repository struct {
	Carts  *mongo.Collection
	Orders *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *Repository {
	return &Repository{
		Carts:  db.Collection("carts"),
		Orders: db.Collection("orders"),
	}
}
