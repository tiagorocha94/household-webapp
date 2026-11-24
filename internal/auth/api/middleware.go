package api

import (
	"log/slog"
	"net/http"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/service"
)

// RequireAuth answers 401 with a JSON error unless the request carries a
// valid session cookie, and otherwise stores the user in the request context.
// The cookie is enough while the frontend is served from the same origin.
func RequireAuth(service *service.Service, log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *auth.User
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			user, err = service.GetUserByToken(cookie.Value)
			if err != nil {
				log.Error("check session", "error", err)
				httpapi.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		}
		if user == nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ctx := auth.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
