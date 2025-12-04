package repository

import (
	"context"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MovieRequestRepository struct {
	col *mongo.Collection
}

func NewMovieRequestRepository() *MovieRequestRepository {
	return &MovieRequestRepository{
		col: db.DB().Collection("movie_requests"),
	}
}

func (r *MovieRequestRepository) Insert(ctx context.Context, mr *models.MovieRequest) error {
	_, err := r.col.InsertOne(ctx, mr)
	return err
}

func (r *MovieRequestRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.MovieRequest, error) {
	var mr models.MovieRequest
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&mr)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &mr, err
}

func (r *MovieRequestRepository) Update(ctx context.Context, mr *models.MovieRequest) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": mr.ID}, mr)
	return err
}

func (r *MovieRequestRepository) FindByUser(
	ctx context.Context,
	userID int,
	status string,
	limit, offset int,
) ([]models.MovieRequest, error) {

	filter := bson.M{"userId": userID}
	if status != "" && status != "all" {
		filter["status"] = status
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.MovieRequest
	for cur.Next(ctx) {
		var mr models.MovieRequest
		if err := cur.Decode(&mr); err != nil {
			return nil, err
		}
		out = append(out, mr)
	}
	return out, cur.Err()
}

func (r *MovieRequestRepository) FindAll(
	ctx context.Context,
	status string,
	limit, offset int,
) ([]models.MovieRequest, error) {

	filter := bson.M{}
	if status != "" && status != "all" {
		filter["status"] = status
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.MovieRequest
	for cur.Next(ctx) {
		var mr models.MovieRequest
		if err := cur.Decode(&mr); err != nil {
			return nil, err
		}
		out = append(out, mr)
	}
	return out, cur.Err()
}
