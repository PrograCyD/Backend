package service

import (
	"context"
	"time"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MovieRequestService struct {
	repo      *repository.MovieRequestRepository
	movieRepo *repository.MovieRepository
	movieSvc  *MovieService
}

func NewMovieRequestService(
	repo *repository.MovieRequestRepository,
	movieRepo *repository.MovieRepository,
	movieSvc *MovieService,
) *MovieRequestService {
	return &MovieRequestService{
		repo:      repo,
		movieRepo: movieRepo,
		movieSvc:  movieSvc,
	}
}

// Crear request (user)
func (s *MovieRequestService) CreateRequest(
	ctx context.Context,
	userID int,
	req *models.MovieCreateRequest,
) (*models.MovieRequest, error) {

	now := time.Now()

	mr := &models.MovieRequest{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Status:    models.MovieRequestStatusPending,
		Movie:     *req,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Insert(ctx, mr); err != nil {
		return nil, err
	}
	return mr, nil
}

func (s *MovieRequestService) ListMine(
	ctx context.Context,
	userID int,
	status string,
	limit, offset int,
) ([]models.MovieRequest, error) {

	return s.repo.FindByUser(ctx, userID, status, limit, offset)
}

func (s *MovieRequestService) ListAll(
	ctx context.Context,
	status string,
	limit, offset int,
) ([]models.MovieRequest, error) {

	return s.repo.FindAll(ctx, status, limit, offset)
}

// Aprobar request: crea película y marca request como approved
func (s *MovieRequestService) Approve(
	ctx context.Context,
	id primitive.ObjectID,
	override *models.MovieCreateRequest,
) (*models.MovieRequest, *models.MovieDoc, error) {

	mr, err := s.repo.FindByID(ctx, id)
	if err != nil || mr == nil {
		return mr, nil, err
	}
	if mr.Status != models.MovieRequestStatusPending {
		return mr, nil, nil // handler puede devolver 400 si no está pending
	}

	// Datos finales de película = request original + override (si viene)
	payload := mr.Movie
	if override != nil {
		if override.Title != "" {
			payload.Title = override.Title
		}
		if override.Year != nil {
			payload.Year = override.Year
		}
		if override.Genres != nil && len(override.Genres) > 0 {
			payload.Genres = override.Genres
		}
		if override.Overview != "" {
			payload.Overview = override.Overview
		}
		if override.Runtime > 0 {
			payload.Runtime = override.Runtime
		}
		if override.Director != "" {
			payload.Director = override.Director
		}
		if override.Cast != nil && len(override.Cast) > 0 {
			payload.Cast = override.Cast
		}
		if override.PosterURL != "" {
			payload.PosterURL = override.PosterURL
		}
		if override.Links != nil {
			payload.Links = override.Links
		}
	}

	// Crear película
	movie, err := s.movieSvc.CreateMovie(ctx, &payload)
	if err != nil {
		return mr, nil, err
	}

	mr.Status = models.MovieRequestStatusApproved
	mr.ApprovedMovieID = &movie.MovieID
	mr.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, mr); err != nil {
		return mr, movie, err
	}

	return mr, movie, nil
}

// Rechazar request
func (s *MovieRequestService) Reject(
	ctx context.Context,
	id primitive.ObjectID,
	reason string,
) (*models.MovieRequest, error) {

	mr, err := s.repo.FindByID(ctx, id)
	if err != nil || mr == nil {
		return mr, err
	}
	if mr.Status != models.MovieRequestStatusPending {
		return mr, nil
	}

	mr.Status = models.MovieRequestStatusRejected
	mr.Reason = reason
	mr.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, mr); err != nil {
		return mr, err
	}
	return mr, nil
}
