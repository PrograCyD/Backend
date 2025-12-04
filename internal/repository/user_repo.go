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

func (r *UserRepository) Search(
	ctx context.Context,
	role, q string,
	limit, offset int,
) ([]models.UserDoc, error) {

	filter := bson.M{}

	if role != "" && role != "all" {
		filter["role"] = role
	}

	if q != "" {
		filter["$or"] = []bson.M{
			{"email": bson.M{"$regex": q, "$options": "i"}},
			{"username": bson.M{"$regex": q, "$options": "i"}},
			{"firstName": bson.M{"$regex": q, "$options": "i"}},
			{"lastName": bson.M{"$regex": q, "$options": "i"}},
		}
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "userId", Value: 1}})

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.UserDoc
	for cur.Next(ctx) {
		var u models.UserDoc
		if err := cur.Decode(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, cur.Err()
}

func (r *UserRepository) GetNextUIdx(ctx context.Context) (*int, error) {
	// buscamos el usuario con uIdx más alto
	opts := options.FindOne().
		SetSort(bson.D{{Key: "uIdx", Value: -1}}).
		SetProjection(bson.M{"uIdx": 1})

	var u models.UserDoc
	err := r.col.FindOne(ctx, bson.M{"uIdx": bson.M{"$ne": nil}}, opts).Decode(&u)
	if err == mongo.ErrNoDocuments {
		// si no hay ninguno con uIdx, empezamos en 0
		x := 0
		return &x, nil
	}
	if err != nil {
		return nil, err
	}
	if u.UIdx == nil {
		x := 0
		return &x, nil
	}
	next := *u.UIdx + 1
	return &next, nil
}
