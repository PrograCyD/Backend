package repository

import (
	"context"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository() *UserRepository {
	return &UserRepository{col: db.DB().Collection("users")}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.UserDoc, error) {
	var u models.UserDoc
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) FindByID(ctx context.Context, userID int) (*models.UserDoc, error) {
	var u models.UserDoc
	err := r.col.FindOne(ctx, bson.M{"userId": userID}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) GetNextUserID(ctx context.Context) (int, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "userId", Value: -1}})
	var u models.UserDoc
	err := r.col.FindOne(ctx, bson.M{}, opts).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return u.UserID + 1, nil
}

func (r *UserRepository) Insert(ctx context.Context, u *models.UserDoc) error {
	_, err := r.col.InsertOne(ctx, u)
	return err
}

// UpdateByID aplica un $set parcial sobre el usuario.
func (r *UserRepository) UpdateByID(ctx context.Context, userID int, update bson.M) error {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"userId": userID},
		bson.M{"$set": update},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
