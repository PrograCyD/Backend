// internal/service/movie_service.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"
)

var ErrMovieAlreadyExists = errors.New("movie already exists")

type MovieService struct {
	movies     *repository.MovieRepository
	tmdbAPIKey string
}

type tmdbKeyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbKeywordsResponse struct {
	Keywords []tmdbKeyword `json:"keywords"`
}

func NewMovieService(m *repository.MovieRepository, tmdbAPIKey string) *MovieService {
	return &MovieService{
		movies:     m,
		tmdbAPIKey: tmdbAPIKey,
	}
}

func (s *MovieService) GetMovie(ctx context.Context, id int) (*models.MovieDoc, error) {
	return s.movies.GetByID(ctx, id)
}

// Crear nueva película (solo admin)
func (s *MovieService) CreateMovie(ctx context.Context, req *models.MovieCreateRequest) (*models.MovieDoc, error) {
	//Validar que no exista ya una película con mismo título + año
	exists, err := s.movies.ExistsByTitleYear(ctx, req.Title, req.Year)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrMovieAlreadyExists
	}
	nextID, err := s.movies.NextMovieID(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)

	md := &models.MovieDoc{
		MovieID:    nextID,
		Title:      req.Title,
		Year:       req.Year,
		Genres:     req.Genres,
		UserTags:   req.UserTags,   // aunque venga vacío, lo guardamos
		GenomeTags: req.GenomeTags, // idem

		// ratingStats inicializado en 0 (sin ratings)
		RatingStats: &models.RatingStats{
			Average:     0,
			Count:       0,
			LastRatedAt: "",
		},

		CreatedAt: now,
		UpdatedAt: now,
	}

	// Links: aseguramos que haya estructura, aunque algunos campos estén vacíos.
	if req.Links != nil {
		md.Links = req.Links
	} else {
		md.Links = &models.Links{
			Movielens: "", // no se conoce, pero lo dejamos explícitamente vacío
		}
	}

	// ExternalData
	if req.Overview != "" ||
		req.Runtime > 0 ||
		req.Director != "" ||
		len(req.Cast) > 0 ||
		req.PosterURL != "" {

		md.ExternalData = &models.ExternalData{
			PosterURL: req.PosterURL,
			Overview:  req.Overview,
			Cast:      req.Cast,
			Director:  req.Director,
			Runtime:   req.Runtime,
			// Budget / Revenue los podrías añadir también al MovieCreateRequest si quieres
		}
	}

	if err := s.movies.Insert(ctx, md); err != nil {
		return nil, err
	}

	return md, nil
}

// Actualizar película existente (solo admin)
func (s *MovieService) UpdateMovie(ctx context.Context, id int, req *models.MovieUpdateRequest) (*models.MovieDoc, error) {
	md, err := s.movies.GetByID(ctx, id)
	if err != nil || md == nil {
		return md, err // si md == nil, handler devuelve 404
	}

	// -------- Campos simples --------
	if req.Title != nil {
		md.Title = *req.Title
	}
	if req.Year != nil {
		md.Year = req.Year
	}
	if req.Genres != nil { // aquí sí permitimos vaciar géneros si viene []
		md.Genres = req.Genres
	}

	// -------- Tags (userTags / genomeTags) --------
	// OJO: MovieUpdateRequest debe tener:
	//   UserTags  []string     `json:"userTags,omitempty"`
	//   GenomeTags []models.GenomeTag `json:"genomeTags,omitempty"`
	if req.UserTags != nil {
		md.UserTags = *req.UserTags
	}
	if req.GenomeTags != nil {
		md.GenomeTags = *req.GenomeTags
	}

	// -------- Links --------
	if req.Links != nil {
		md.Links = req.Links
	} else if md.Links == nil {
		// aseguramos que haya estructura aunque no se envíe nada
		md.Links = &models.Links{
			Movielens: "",
		}
	}

	// -------- ExternalData --------
	if md.ExternalData == nil {
		md.ExternalData = &models.ExternalData{}
	}
	if req.Overview != nil {
		md.ExternalData.Overview = *req.Overview
	}
	if req.Runtime != nil {
		md.ExternalData.Runtime = *req.Runtime
	}
	if req.Director != nil {
		md.ExternalData.Director = *req.Director
	}
	if req.Cast != nil {
		md.ExternalData.Cast = req.Cast
	}
	if req.PosterURL != nil {
		md.ExternalData.PosterURL = *req.PosterURL
	}

	// -------- RatingStats --------
	// No los reseteamos, solo nos aseguramos de que no sean nil.
	if md.RatingStats == nil {
		md.RatingStats = &models.RatingStats{
			Average:     0,
			Count:       0,
			LastRatedAt: "",
		}
	}

	md.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.movies.Update(ctx, md); err != nil {
		return nil, err
	}
	return md, nil
}

func (s *MovieService) Search(
	ctx context.Context,
	q, genre string,
	yearFrom, yearTo, limit, offset int,
) ([]models.MovieDoc, error) {
	return s.movies.Search(ctx, q, genre, yearFrom, yearTo, limit, offset)
}

func (s *MovieService) Top(ctx context.Context, metric string, limit int) ([]models.MovieDoc, error) {
	return s.movies.Top(ctx, metric, limit)
}

// ==================== TMDB =====================

// estructuras mínimas para parsear la respuesta de TMDB
type tmdbMovieResponse struct {
	Title       string `json:"title"`
	Overview    string `json:"overview"`
	Runtime     int    `json:"runtime"`
	Budget      int    `json:"budget"`
	Revenue     int64  `json:"revenue"`
	PosterPath  string `json:"poster_path"`
	ReleaseDate string `json:"release_date"`
	ImdbID      string `json:"imdb_id"`
	ID          int    `json:"id"`

	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

type tmdbCastMember struct {
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path"`
}

type tmdbCrewMember struct {
	Name string `json:"name"`
	Job  string `json:"job"`
}

type tmdbCreditsResponse struct {
	Cast []tmdbCastMember `json:"cast"`
	Crew []tmdbCrewMember `json:"crew"`
}

// FetchExternalFromTMDB obtiene los datos "ExternalData" de TMDB
// a partir de un tmdbId (string).
func (s *MovieService) FetchExternalFromTMDB(ctx context.Context, tmdbID string) (*models.ExternalData, error) {
	if s.tmdbAPIKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY no configurado")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 1) Detalles de la película
	movieURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s?api_key=%s", tmdbID, s.tmdbAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, movieURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al llamar a TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// no encontrada, devolvemos ExternalData vacío
		return &models.ExternalData{TMDBFetched: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB devolvió status %d", resp.StatusCode)
	}

	var movieData tmdbMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movieData); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de TMDB: %w", err)
	}

	// 2) Créditos (cast + director)
	creditsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s/credits?api_key=%s", tmdbID, s.tmdbAPIKey)
	creditsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, creditsURL, nil)
	if err != nil {
		return nil, err
	}

	creditsResp, err := client.Do(creditsReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a TMDB credits: %w", err)
	}
	defer creditsResp.Body.Close()

	var creditsData tmdbCreditsResponse
	if creditsResp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(creditsResp.Body).Decode(&creditsData); err != nil {
			creditsData = tmdbCreditsResponse{}
		}
	}

	// 3) Construimos ExternalData
	ext := &models.ExternalData{
		Overview:    movieData.Overview,
		Runtime:     movieData.Runtime,
		Budget:      movieData.Budget,
		Revenue:     movieData.Revenue,
		TMDBFetched: true,
	}

	if movieData.PosterPath != "" {
		ext.PosterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", movieData.PosterPath)
	}

	// top 10 del cast
	maxCast := 10
	for i, member := range creditsData.Cast {
		if i >= maxCast {
			break
		}
		castMember := models.CastMember{Name: member.Name}
		if member.ProfilePath != "" {
			castMember.ProfileURL = fmt.Sprintf("https://image.tmdb.org/t/p/w185%s", member.ProfilePath)
		}
		ext.Cast = append(ext.Cast, castMember)
	}

	// director
	for _, member := range creditsData.Crew {
		if member.Job == "Director" {
			ext.Director = member.Name
			break
		}
	}

	return ext, nil
}

// PrefillCreateFromTMDB construye un MovieCreateRequest casi completo
// a partir de un tmdbId, usando la misma info de TMDB.
func (s *MovieService) PrefillCreateFromTMDB(ctx context.Context, tmdbID string) (*models.MovieCreateRequest, error) {
	if s.tmdbAPIKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY no configurado")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 1) Detalles de la película
	movieURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s?api_key=%s", tmdbID, s.tmdbAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, movieURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al llamar a TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("película no encontrada en TMDB")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB devolvió status %d", resp.StatusCode)
	}

	var movieData tmdbMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movieData); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de TMDB: %w", err)
	}

	// 2) Créditos (cast + director)
	creditsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s/credits?api_key=%s", tmdbID, s.tmdbAPIKey)
	creditsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, creditsURL, nil)
	if err != nil {
		return nil, err
	}

	creditsResp, err := client.Do(creditsReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a TMDB credits: %w", err)
	}
	defer creditsResp.Body.Close()

	var creditsData tmdbCreditsResponse
	if creditsResp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(creditsResp.Body).Decode(&creditsData); err != nil {
			creditsData = tmdbCreditsResponse{}
		}
	}

	// 3) Keywords (para userTags / genomeTags)
	keywordsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s/keywords?api_key=%s", tmdbID, s.tmdbAPIKey)
	keywordsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, keywordsURL, nil)
	if err != nil {
		return nil, err
	}
	keywordsResp, err := client.Do(keywordsReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a TMDB keywords: %w", err)
	}
	defer keywordsResp.Body.Close()

	var keywordsData tmdbKeywordsResponse
	if keywordsResp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(keywordsResp.Body).Decode(&keywordsData); err != nil {
			keywordsData = tmdbKeywordsResponse{}
		}
	}

	// 4) Year desde release_date (YYYY-MM-DD)
	var yearPtr *int
	if movieData.ReleaseDate != "" && len(movieData.ReleaseDate) >= 4 {
		if y, err := strconv.Atoi(movieData.ReleaseDate[:4]); err == nil {
			yearPtr = &y
		}
	}

	// 5) Géneros como slice de string
	var genres []string
	for _, g := range movieData.Genres {
		genres = append(genres, g.Name)
	}

	// 6) Links
	links := &models.Links{
		Movielens: "", // no la conocemos, la dejamos explícita vacía
	}
	if movieData.ID != 0 {
		links.TMDB = fmt.Sprintf("https://www.themoviedb.org/movie/%d", movieData.ID)
	}
	if movieData.ImdbID != "" {
		links.IMDB = fmt.Sprintf("http://www.imdb.com/title/%s/", movieData.ImdbID)
	}

	// 7) Cast (top 10)
	maxCast := 10
	var cast []models.CastMember
	for i, member := range creditsData.Cast {
		if i >= maxCast {
			break
		}
		cm := models.CastMember{Name: member.Name}
		if member.ProfilePath != "" {
			cm.ProfileURL = fmt.Sprintf("https://image.tmdb.org/t/p/w185%s", member.ProfilePath)
		}
		cast = append(cast, cm)
	}

	// 8) Tags a partir de keywords
	var userTags []string
	var genomeTags []models.GenomeTag
	for _, kw := range keywordsData.Keywords {
		userTags = append(userTags, kw.Name)
		// Le ponemos una relevancia fija (p.e. 1.0) porque TMDB no da score
		genomeTags = append(genomeTags, models.GenomeTag{
			Tag:       kw.Name,
			Relevance: 1.0,
		})
	}

	// 9) Armamos el MovieCreateRequest
	reqOut := &models.MovieCreateRequest{
		Title:      movieData.Title,
		Year:       yearPtr,
		Genres:     genres,
		Overview:   movieData.Overview,
		Runtime:    movieData.Runtime,
		Director:   "", // rellenamos abajo con crew
		Cast:       cast,
		PosterURL:  "",
		Links:      links,
		UserTags:   userTags,
		GenomeTags: genomeTags,
	}

	// Director
	for _, m := range creditsData.Crew {
		if m.Job == "Director" {
			reqOut.Director = m.Name
			break
		}
	}

	// Poster
	if movieData.PosterPath != "" {
		reqOut.PosterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", movieData.PosterPath)
	}

	return reqOut, nil
}
