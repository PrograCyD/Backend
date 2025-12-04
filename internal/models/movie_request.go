package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Payload para crear una película (lo que expondremos en la API)
type MovieCreateRequest struct {
	Title  string   `json:"title"` // obligatorio
	Year   *int     `json:"year,omitempty"`
	Genres []string `json:"genres,omitempty"`

	// ExternalData
	Overview   string       `json:"overview,omitempty"`  // mapea a ExternalData.Overview
	Runtime    int          `json:"runtime,omitempty"`   // ExternalData.Runtime
	Director   string       `json:"director,omitempty"`  // ExternalData.Director
	Cast       []CastMember `json:"cast,omitempty"`      // ExternalData.Cast
	PosterURL  string       `json:"posterUrl,omitempty"` // ExternalData.PosterURL
	Links      *Links       `json:"links,omitempty"`
	UserTags   []string     `json:"userTags,omitempty"`
	GenomeTags []GenomeTag  `json:"genomeTags,omitempty"`
}

// Payload para actualización parcial de película
type MovieUpdateRequest struct {
	Title      *string      `json:"title,omitempty"`
	Year       *int         `json:"year,omitempty"`
	Genres     []string     `json:"genres,omitempty"`
	Overview   *string      `json:"overview,omitempty"`
	Runtime    *int         `json:"runtime,omitempty"`
	Director   *string      `json:"director,omitempty"`
	Cast       []CastMember `json:"cast,omitempty"`
	PosterURL  *string      `json:"posterUrl,omitempty"`
	Links      *Links       `json:"links,omitempty"`
	UserTags   *[]string    `json:"userTags,omitempty"`
	GenomeTags *[]GenomeTag `json:"genomeTags,omitempty"`
}

// Estados posibles del request
const (
	MovieRequestStatusPending  = "pending"
	MovieRequestStatusApproved = "approved"
	MovieRequestStatusRejected = "rejected"
)

// Documento para la colección movie_requests
type MovieRequest struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          int                `json:"userId" bson:"userId"`
	Status          string             `json:"status" bson:"status"` // pending|approved|rejected
	Movie           MovieCreateRequest `json:"movie" bson:"movie"`
	ApprovedMovieID *int               `json:"approvedMovieId,omitempty" bson:"approvedMovieId,omitempty"`
	Reason          string             `json:"reason,omitempty" bson:"reason,omitempty"`
	CreatedAt       time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Body para rechazar un request de película.
type RejectMovieRequest struct {
	Reason string `json:"reason"`
}

// TMDBFetchRequest sirve para pedir datos a partir de un id de TMDB.
type TMDBFetchRequest struct {
	TMDBID string `json:"tmdbId" example:"603"` // p.e. "603" para The Matrix
}
