package service

import (
	"context"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"
)

type RatingService struct {
	ratings *repository.RatingRepository
}

func NewRatingService(r *repository.RatingRepository) *RatingService {
	return &RatingService{ratings: r}
}

func (s *RatingService) AddOrUpdate(ctx context.Context, userID, movieID int, rating float64) error {
	return s.ratings.UpsertRating(ctx, userID, movieID, rating)
}

func (s *RatingService) GetByUser(ctx context.Context, userID, limit, offset int) ([]models.RatingDoc, error) {
	return s.ratings.GetByUser(ctx, userID, limit, offset)
}
