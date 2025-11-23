package models

// ----- SUMMARY -----

// AdminSimilaritySummary representa el resumen general de mapeos y similitudes.
type AdminSimilaritySummary struct {
	TotalMovies               int64 `json:"totalMovies"`
	MoviesWithIdx             int64 `json:"moviesWithIdx"`
	MoviesWithoutIdx          int64 `json:"moviesWithoutIdx"`
	MoviesWithSimilarities    int64 `json:"moviesWithSimilarities"`
	MoviesWithoutSimilarities int64 `json:"moviesWithoutSimilarities"`
	MinRatings                int64 `json:"minRatings"`
}

// ----- PENDING -----

// PendingMovieWithoutIdx película sin iIdx pero con suficientes ratings.
type PendingMovieWithoutIdx struct {
	MovieID      int    `json:"movieId"`
	Title        string `json:"title"`
	RatingsCount int64  `json:"ratingsCount"`
}

// PendingMovieWithoutSims película con iIdx pero sin documento en similarities.
type PendingMovieWithoutSims struct {
	MovieID      int    `json:"movieId"`
	IIdx         int    `json:"iIdx"`
	Title        string `json:"title"`
	RatingsCount int64  `json:"ratingsCount"`
}

// AdminPendingSimilarities respuesta de /pending.
type AdminPendingSimilarities struct {
	MinRatings          int64                     `json:"minRatings"`
	WithoutIdx          []PendingMovieWithoutIdx  `json:"withoutIdx"`
	WithoutSimilarities []PendingMovieWithoutSims `json:"withoutSimilarities"`
}

// ----- REMAP MISSING -----

// RemapMissingRequest body de /remap-missing.
type RemapMissingRequest struct {
	MinRatings int64 `json:"minRatings"`
	Limit      int64 `json:"limit"`
}

// RemapMissingResult resultado de /remap-missing.
type RemapMissingResult struct {
	MappedCount int64 `json:"mappedCount"`
	FromIdx     int   `json:"fromIdx"`
	ToIdx       int   `json:"toIdx"`
}

// ----- REBUILD SIMILARITIES -----

// RebuildSimilaritiesRequest body de /rebuild.
type RebuildSimilaritiesRequest struct {
	MinRatings     int64 `json:"minRatings"`
	BatchSize      int   `json:"batchSize"`
	Parallelism    int   `json:"parallelism"`
	K              int   `json:"k"`
	MinCommonUsers int   `json:"minCommonUsers"`
	Shrink         int   `json:"shrink"`
}

// RebuildSimilaritiesResult resultado de /rebuild.
type RebuildSimilaritiesResult struct {
	ProcessedMovies int `json:"processedMovies"`
	Batches         int `json:"batches"`
	K               int `json:"k"`
	MinCommonUsers  int `json:"minCommonUsers"`
	Shrink          int `json:"shrink"`
}
