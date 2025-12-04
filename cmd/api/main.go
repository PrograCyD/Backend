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
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	// Mongo y Redis
	db.InitMongo(cfg)
	cache.InitRedis(cfg)

	// repos
	userRepo := repository.NewUserRepository()
	movieRepo := repository.NewMovieRepository()
	movieReqRepo := repository.NewMovieRequestRepository()
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
	movieSvc := service.NewMovieService(movieRepo, cfg.TMDBAPIKey)
	movieReqSvc := service.NewMovieRequestService(movieReqRepo, movieRepo, movieSvc)
	ratingSvc := service.NewRatingService(ratingRepo, movieRepo)
	// coordinador que habla con los nodos ML + guarda historial + explicaciones
	recSvc := service.NewRecommendService(ratingRepo, recRepo, simRepo, mlNodes)
	// servicio de mantenimiento admin
	adminMaintSvc := service.NewAdminMaintenanceService(cfg, mlNodes)

	// handlers
	authH := handler.NewAuthHandler(authSvc)
	movieH := handler.NewMovieHandler(movieSvc)
	movieReqH := handler.NewMovieRequestHandler(movieReqSvc)
	ratingH := handler.NewRatingHandler(ratingSvc)
	recH := handler.NewRecommendHandler(recSvc)
	adminMaintH := handler.NewAdminMaintenanceHandler(adminMaintSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// =============
	// Rutas públicas
	// =============
	r.Get("/health", handler.Health)

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	// Películas (públicas)
	r.Get("/movies/tmdb", movieH.FetchFromTMDB)
	r.Get("/movies/tmdb-prefill", movieH.PrefillMovieFromTMDB)
	r.Get("/movies/{id}", movieH.GetMovie)
	r.Get("/movies/search", movieH.Search)
	r.Get("/movies/top", movieH.Top)

	// ===========================
	// Rutas protegidas con JWT
	// ===========================
	authMw := handler.JWTAuth(cfg.JWTSecret)

	r.Group(func(r chi.Router) {
		r.Use(authMw)

		// ---- Endpoints /me (USER normal) ----
		r.Route("/me", func(r chi.Router) {
			r.Get("/ratings", ratingH.GetMyRatings)
			r.Post("/ratings", ratingH.PostMyRating)
			r.Get("/recommendations", recH.GetMyRecommendations)

			// movie requests (USER)
			r.Get("/movie-requests", movieReqH.ListMine)
			r.Post("/movie-requests", movieReqH.Create)
		})

		// ---- Endpoints solo ADMIN ----
		r.Group(func(r chi.Router) {
			r.Use(handler.AdminOnly())

			// edición de usuario
			r.Put("/users/{id}/update", authH.UpdateUser)

			// gestión de películas
			r.Post("/admin/movies", movieH.CreateMovie)
			r.Put("/admin/movies/{id}", movieH.UpdateMovie)
			r.Get("/users", authH.ListUsers)

			// ratings y recomendaciones de cualquier usuario
			r.Route("/users/{id}", func(r chi.Router) {
				// obtener info del usuario por id
				r.Get("/", authH.GetUserByID)

				r.Get("/ratings", ratingH.GetRatings)
				r.Post("/ratings", ratingH.PostRating)

				// HTTP normal
				r.Get("/recommendations", recH.GetRecommendations)

				// WebSocket
				r.Get("/ws/recommendations", recH.GetRecommendationsWS)
			})

			// movie-requests (ADMIN)
			r.Get("/admin/movie-requests", movieReqH.ListAll)
			r.Post("/admin/movie-requests/{id}/approve", movieReqH.Approve)
			r.Post("/admin/movie-requests/{id}/reject", movieReqH.Reject)

			// --- mantenimiento de similitudes / mapeos ---
			handler.MountAdminMaintenanceRoutes(r, adminMaintH)
		})
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	log.Printf("HTTP escuchando en :%s", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, r))
}
