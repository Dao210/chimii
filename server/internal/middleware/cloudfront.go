package middleware

import (
	"net/http"
	"time"

	"github.com/chimii-ai/chimii/server/internal/auth"
)

// RefreshCloudFrontCookies is middleware that refreshes CloudFront signed cookies
// on authenticated requests when the cookie is missing (expired or first request
// after login). This prevents 403s from the CDN when cookies expire before the
// user's session does.
func RefreshCloudFrontCookies(signer *auth.CloudFrontSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if signer == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// A child session must not inherit the parent's broad attachment CDN
			// authorization from cookies already present in the browser. Expire all
			// three signed cookies before serving the restricted child surface.
			if r.Header.Get("X-Actor-Source") == "child_session" {
				for _, cookie := range signer.SignedCookies(time.Now()) {
					cookie.Value = ""
					cookie.Expires = time.Unix(1, 0)
					cookie.MaxAge = -1
					http.SetCookie(w, cookie)
				}
				next.ServeHTTP(w, r)
				return
			}
			if _, err := r.Cookie("CloudFront-Policy"); err != nil {
				ttl := auth.AuthTokenTTL()
				for _, cookie := range signer.SignedCookies(time.Now().Add(ttl)) {
					http.SetCookie(w, cookie)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
