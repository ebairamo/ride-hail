package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"ride-hail/internal/core/domain/action"
	"ride-hail/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

func (a *API) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = newRequestID()
		}

		w.Header().Set("X-Request-ID", reqID)
		ctx := logger.WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := a.log.Func("api.jwtMiddleware")

		// Получение cookie
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				log.Warn(r.Context(), action.Authorization, "no authorization cookie")
				http.Error(w, "no authorization cookie", http.StatusUnauthorized)
				return
			}
			log.Error(r.Context(), action.Authorization, "error getting authorization cookie", "error", err)
			http.Error(w, "cookie error", http.StatusBadRequest)
			return
		}

		token := cookie.Value
		if token == "" {
			log.Warn(r.Context(), action.Authorization, "no token provided")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// Парсинг JWT токена
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.cfg.JWT.Secret), nil
		})
		if err != nil {
			log.Warn(r.Context(), action.Authorization, "failed to parse token", "error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !t.Valid {
			log.Warn(r.Context(), action.Authorization, "token is invalid")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// Извлечение claims
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			log.Warn(r.Context(), action.Authorization, "invalid claims type")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// Валидация user_id и role
		userID, okID := claims["user_id"].(string)
		role, okRole := claims["role"].(string)

		if !okID || !okRole {
			log.Warn(r.Context(), action.Authorization, "invalid user_id or role in claims",
				"has_user_id", okID,
				"has_role", okRole,
			)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if userID == "" || role == "" {
			log.Warn(r.Context(), action.Authorization, "empty user_id or role")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// Обогащение контекста
		ctx := logger.WithUserID(r.Context(), userID)
		ctx = logger.WithRole(ctx, role)

		log.Debug(r.Context(), action.Authorization, "user authorized",
			"user_id", userID,
			"role", role,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback на timestamp если не удалось сгенерировать random
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}
