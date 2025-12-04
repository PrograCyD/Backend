package service

import (
	"context"
	"fmt"
	"time"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"
)

type RatingService struct {
	ratings *repository.RatingRepository
	movies  *repository.MovieRepository
}

func NewRatingService(r *repository.RatingRepository, m *repository.MovieRepository) *RatingService {
	return &RatingService{
		ratings: r,
		movies:  m,
	}
}

func (s *RatingService) AddOrUpdate(ctx context.Context, userID, movieID int, rating float64) error {
	// 1) Ver si ya existía un rating previo
	prev, err := s.ratings.GetOne(ctx, userID, movieID)
	if err != nil {
		return err
	}
	existedBefore := prev != nil

	// 2) Upsert del rating (guarda timestamp como epoch)
	if err := s.ratings.UpsertRating(ctx, userID, movieID, rating); err != nil {
		return err
	}

	// 3) Actualizar stats de la película
	movie, err := s.movies.GetByID(ctx, movieID)
	if err != nil {
		return err
	}
	if movie == nil {
		// si quieres podrías devolver error más explícito
		return fmt.Errorf("movie %d no encontrada", movieID)
	}

	// Aseguramos estructura de ratingStats
	if movie.RatingStats == nil {
		movie.RatingStats = &models.RatingStats{
			Average: 0,
			Count:   0,
		}
	}
	rs := movie.RatingStats

	// count siempre > 0 si ya hay ratings; usamos fórmulas en float64
	if !existedBefore {
		// Nuevo rating
		total := rs.Average*float64(rs.Count) + rating
		rs.Count++
		if rs.Count > 0 {
			rs.Average = total / float64(rs.Count)
		}
	} else {
		// Update de rating existente
		total := rs.Average*float64(rs.Count) - prev.Rating + rating
		if rs.Count > 0 {
			rs.Average = total / float64(rs.Count)
		}
		// rs.Count no cambia
	}

	nowStr := time.Now().Format(time.RFC3339)
	rs.LastRatedAt = nowStr
	movie.UpdatedAt = nowStr

	return s.movies.Update(ctx, movie)
}

func (s *RatingService) GetByUser(ctx context.Context, userID, limit, offset int) ([]models.RatingDoc, error) {
	return s.ratings.GetByUser(ctx, userID, limit, offset)
}
