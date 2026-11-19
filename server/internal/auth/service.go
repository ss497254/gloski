// Package auth validates the optional API key and JWT credentials used by the
// browser-facing master API.
package auth

import (
	"crypto/rsa"
	"crypto/subtle"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ss497254/gloski/internal/config"
)

// Errors returned by credential validation.
var (
	// ErrInvalidAPIKey means the supplied API key does not match the configured one.
	ErrInvalidAPIKey = errors.New("invalid API key")
	// ErrInvalidToken means no JWT public key is configured or the token is malformed.
	ErrInvalidToken = errors.New("invalid token")
	// ErrNoAuthMethod means no API key or JWT public key is configured.
	ErrNoAuthMethod = errors.New("no authentication method configured")
	// ErrTokenExpired means the JWT's exp claim is in the past.
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidAudience means the JWT's audience does not include the master.
	ErrInvalidAudience = errors.New("invalid audience")
)

// Service checks the credentials accepted by the browser-facing API.
type Service struct {
	apiKey       string
	jwtPublicKey *rsa.PublicKey
}

// NewService builds the credential checker for the configured API key and JWT
// public key.
func NewService(cfg *config.Config) (*Service, error) {
	s := &Service{
		apiKey: cfg.APIKey,
	}

	// Parse JWT public key if provided
	if cfg.JWTPublicKey != "" {
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.JWTPublicKey))
		if err != nil {
			return nil, errors.New("failed to parse JWT public key: " + err.Error())
		}
		s.jwtPublicKey = key
	}

	return s, nil
}

// ValidateAPIKey checks if the provided API key is valid
func (s *Service) ValidateAPIKey(key string) error {
	if s.apiKey == "" {
		return ErrInvalidAPIKey
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(s.apiKey)) != 1 {
		return ErrInvalidAPIKey
	}
	return nil
}

// ValidateToken validates a JWT token using the configured public key
func (s *Service) ValidateToken(tokenString string) error {
	if s.jwtPublicKey == nil {
		return ErrInvalidToken
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtPublicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrTokenExpired
		}
		return ErrInvalidToken
	}

	if !token.Valid {
		return ErrInvalidToken
	}

	return nil
}

// HasAPIKey returns true if API key authentication is configured
func (s *Service) HasAPIKey() bool {
	return s.apiKey != ""
}

// HasJWT returns true if JWT authentication is configured
func (s *Service) HasJWT() bool {
	return s.jwtPublicKey != nil
}
