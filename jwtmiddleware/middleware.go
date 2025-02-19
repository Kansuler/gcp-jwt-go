package jwtmiddleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang-jwt/jwt/v4/request"

	gcpjwt "github.com/Kansuler/gcp-jwt-go/v2"
)

// NewHandler will return a middleware that will try and validate tokens in incoming HTTP requests.
// The token is expected as a Bearer token in the Authorization header and expected to have an Issuer
// claim equal to the ServiceAccount the provided IAMConfig is configured for. This will also validate the
// Audience claim to the one provided, or use https:// + request.Host if blank. NOTE: If using the signJwt method,
// you MUST call gcpjwt.SigningMethodIAMJWT.Override().
//
// Complimentary to https://github.com/Kansuler/gcp-jwt-go/oauth2
func NewHandler(ctx context.Context, config *gcpjwt.IAMConfig, audience string) func(http.Handler) http.Handler {
	ctx = gcpjwt.NewIAMContext(ctx, config)

	keyFunc := gcpjwt.IAMVerfiyKeyfunc(ctx, config)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &jwt.StandardClaims{}

			token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, keyFunc, request.WithClaims(claims))
			if err != nil || !token.Valid {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			aud := audience
			if aud == "" {
				aud = fmt.Sprintf("https://%s", r.Host)
			}

			if !claims.VerifyAudience(aud, true) || !claims.VerifyIssuer(config.ServiceAccount, true) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
