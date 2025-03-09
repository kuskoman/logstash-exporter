package tls

import (
	"net/http"
)

// MultiUserAuthMiddleware adds basic authentication with multiple users to an HTTP handler.
func MultiUserAuthMiddleware(next http.Handler, users map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+BasicAuthRealm+`"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		storedPassword, userExists := users[user]
		if !userExists || storedPassword != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+BasicAuthRealm+`"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
