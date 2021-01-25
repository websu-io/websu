package api

import (
	"context"
	"errors"
	firebase "firebase.google.com/go/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

var FirebaseApp *firebase.App

func InitFirebase() {
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.WithError(err).Fatal("Error initializing firebase")
	}
	FirebaseApp = app
}

func verifyIDToken(idToken string) (string, error) {
	ctx := context.Background()
	client, err := FirebaseApp.Auth(ctx)
	if err != nil {
		return "", err
	}
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	return token.UID, nil
}

func TokenFromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}

	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

func JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := TokenFromAuthHeader(r)
		if err != nil {
			log.WithError(err).Error("Error parsing Authentication header")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if token != "" {
			userID, err := verifyIDToken(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			log.WithField("user", userID).Info("User provided a valid token")
			ctx := context.WithValue(r.Context(), "UserID", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
