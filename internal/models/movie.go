package models

type Links struct {
	Movielens string `json:"movielens,omitempty" bson:"movielens,omitempty"`
	IMDB      string `json:"imdb,omitempty" bson:"imdb,omitempty"`
	TMDB      string `json:"tmdb,omitempty" bson:"tmdb,omitempty"`
}

type GenomeTag struct {
	Tag       string  `json:"tag" bson:"tag"`
	Relevance float64 `json:"relevance" bson:"relevance"`
}

type CastMember struct {
	Name       string `json:"name" bson:"name"`
	ProfileURL string `json:"profileUrl,omitempty" bson:"profileUrl,omitempty"`
}

type ExternalData struct {
	PosterURL   string       `json:"posterUrl,omitempty" bson:"posterUrl,omitempty"`
	Overview    string       `json:"overview,omitempty" bson:"overview,omitempty"`
	Cast        []CastMember `json:"cast,omitempty" bson:"cast,omitempty"`
	Director    string       `json:"director,omitempty" bson:"director,omitempty"`
	Runtime     int          `json:"runtime,omitempty" bson:"runtime,omitempty"`
	Budget      int          `json:"budget,omitempty" bson:"budget,omitempty"`
	Revenue     int64        `json:"revenue,omitempty" bson:"revenue,omitempty"`
	TMDBFetched bool         `json:"tmdbFetched" bson:"tmdbFetched"`
}

type RatingStats struct {
	Average     float64 `json:"average" bson:"average"`
	Count       int     `json:"count" bson:"count"`
	LastRatedAt string  `json:"lastRatedAt,omitempty" bson:"lastRatedAt,omitempty"`
}

type MovieDoc struct {
	MovieID      int           `json:"movieId" bson:"movieId"`
	IIdx         *int          `json:"iIdx,omitempty" bson:"iIdx,omitempty"`
	Title        string        `json:"title" bson:"title"`
	Year         *int          `json:"year,omitempty" bson:"year,omitempty"`
	Genres       []string      `json:"genres" bson:"genres"`
	Links        *Links        `json:"links,omitempty" bson:"links,omitempty"`
	GenomeTags   []GenomeTag   `json:"genomeTags,omitempty" bson:"genomeTags,omitempty"`
	UserTags     []string      `json:"userTags,omitempty" bson:"userTags,omitempty"`
	RatingStats  *RatingStats  `json:"ratingStats,omitempty" bson:"ratingStats,omitempty"`
	ExternalData *ExternalData `json:"externalData,omitempty" bson:"externalData,omitempty"`
	CreatedAt    string        `json:"createdAt" bson:"createdAt"`
	UpdatedAt    string        `json:"updatedAt" bson:"updatedAt"`
}
