package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID   ctxKey = "userId"
	CtxUserRole ctxKey = "role"
)

// JWTAuth devuelve un middleware que valida el token JWT y
// mete userId y role en el contexto.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secretBytes, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			subVal, ok := claims["sub"].(float64)
			if !ok {
				http.Error(w, "invalid sub in token", http.StatusUnauthorized)
				return
			}
			role, _ := claims["role"].(string)

			ctx := context.WithValue(r.Context(), CtxUserID, int(subVal))
			ctx = context.WithValue(ctx, CtxUserRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly solo deja pasar a role == "admin".
func AdminOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxUserRole).(string)
			if role != "admin" {
				http.Error(w, "admin only", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext helper para sacar el userId del contexto.
func UserIDFromContext(ctx context.Context) int {
	if v := ctx.Value(CtxUserID); v != nil {
		if id, ok := v.(int); ok {
			return id
		}
	}
	return 0
}
