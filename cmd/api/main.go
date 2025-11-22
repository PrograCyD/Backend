package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	_ "nodosml-pc4/docs" // swagger docs

	"nodosml-pc4/internal/cache"
	"nodosml-pc4/internal/config"
	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/handler"
	"nodosml-pc4/internal/repository"
	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title NodosML Movie Recommender API
// @version 1.0
// @description API para PC4 (item-based, Mongo, Redis)
// @host localhost:8080
// @BasePath /
func main() {
	cfg := config.Load()

	// Mongo y Redis
	db.InitMongo(cfg)
	cache.InitRedis(cfg)

	// repos
	userRepo := repository.NewUserRepository()
	movieRepo := repository.NewMovieRepository()
	ratingRepo := repository.NewRatingRepository()
	recRepo := repository.NewRecommendationRepository()
	simRepo := repository.NewSimilarityRepository()

	// ============================
	// Leer direcciones de nodos ML
	// ============================
	var mlNodes []string
	if env := os.Getenv("ML_NODE_ADDRS"); env != "" {
		for _, v := range strings.Split(env, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				mlNodes = append(mlNodes, v)
			}
		}
	}

	// fallback por si no hay variable de entorno (útil en local sin Docker)
	if len(mlNodes) == 0 {
		mlNodes = []string{
			"mlnode1:9001",
			"mlnode2:9001",
			"mlnode3:9001",
			"mlnode4:9001",
		}
	}

	// services
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	movieSvc := service.NewMovieService(movieRepo)
	ratingSvc := service.NewRatingService(ratingRepo)
	// coordinador que habla con los nodos ML + guarda historial + explicaciones
	recSvc := service.NewRecommendService(ratingRepo, recRepo, simRepo, mlNodes)

	// handlers
	authH := handler.NewAuthHandler(authSvc)
	movieH := handler.NewMovieHandler(movieSvc)
	ratingH := handler.NewRatingHandler(ratingSvc)
	recH := handler.NewRecommendHandler(recSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// rutas públicas
	r.Get("/health", handler.Health)

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)
	r.Put("/users/{id}/update", authH.UpdateUser)

	// películas
	r.Get("/movies/{id}", movieH.GetMovie)
	r.Get("/movies/search", movieH.Search)

	// rutas de usuarios
	r.Route("/users", func(r chi.Router) {
		// PUT /users/{id} -> actualizar usuario
		// r.Put("/{id}", authH.UpdateUser)

		// rutas que dependen de userId
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/ratings", ratingH.GetRatings)
			r.Post("/ratings", ratingH.PostRating)

			// HTTP normal
			r.Get("/recommendations", recH.GetRecommendations)

			// WebSocket
			r.Get("/ws/recommendations", recH.GetRecommendationsWS)
		})
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	log.Printf("HTTP escuchando en :%s", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, r))
}
