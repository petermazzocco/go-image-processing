package auth

import (
	"context"
	"log"
	"net/http"

	"github.com/markbates/goth/gothic"
)

func UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		q.Set("provider", "google")
		r.URL.RawQuery = q.Encode()

		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			log.Println("User not authenticated:", err)
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", user.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
