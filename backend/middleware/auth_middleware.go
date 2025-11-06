package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/auth"
)

// JwtMiddleware protects routes by checking for a valid JWT.
// It checks for a token in two places:
// 1. The "Authorization: Bearer <token>" header (for standard HTTP requests)
// 2. A "token=<token>" query parameter (for WebSocket connections)
func JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string // This variable will hold our token

		// 1. Try to get the token from the Authorization header
		// This is for your standard HTTP requests (POST, GET)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. If the header was empty, try to get the token from the URL
		// This is the FIX for your WebSocket connection
		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}

		// 3. If the token is *still* empty after both checks,
		// it means no token was provided at all. We must reject the request.
		if tokenString == "" {
			http.Error(w, "Forbidden: No token provided", http.StatusForbidden)
			return
		}

		// 4. We found a token (from one of the two places). Now, validate it.
		claims, err := auth.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Forbidden: Invalid token", http.StatusForbidden)
			return
		}

		// 5. The token is valid! Store the user's info in the context
		// for the next handler (like CreateRoom or HandleConnections).
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)

		// 6. Call the next handler in the chain, passing the updated context.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
