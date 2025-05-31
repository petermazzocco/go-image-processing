package auth

import (
	"context"
	"net/http"

	"github.com/markbates/goth/gothic"
)

func UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := gothic.Store.Get(r, "gothic_session")
		if err != nil {
			http.Error(w, "Not Authorized", http.StatusInternalServerError)
			return
		}

		userID, ok := session.Values["user_id"]
		if !ok || userID == "" {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
