package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prophet7821/speedBump.git"
	"net/http"
	"time"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/admin", func(r chi.Router) {

		//Setting Context for userID
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "userID", "123")))
			})
		})

		r.Use(speedBump.Limit(
			0,
			10*time.Second,
			speedBump.WithKeyFuncs(speedBump.KeyByIP, func(r *http.Request) (string, error) {
				token := r.Context().Value("userID").(string)
				return token, nil
			}),

			speedBump.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, err := w.Write([]byte(`{"message": "Too many requests"}`))
				if err != nil {
					return
				}
			}),
		))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("admin"))
			if err != nil {
				return
			}
		})

	})

	r.Group(func(r chi.Router) {
		r.Use(speedBump.LimitByIP(0, 10*time.Second))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("home"))
			if err != nil {
				return
			}
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(speedBump.Limit(0, 10*time.Second, speedBump.WithRedisLimitCounter(&speedBump.Config{})))
		r.Get("/redis", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("home"))
			if err != nil {
				return
			}
		})
	})

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}

}
