package auth

import (
	"crypto/rsa"
	"crypto/subtle"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ss497254/gloski/internal/config"
)

var (
	ErrInvalidAPIKey   = errors.New("invalid API key")
	ErrInvalidToken    = errors.New("invalid token")
	ErrNoAuthMethod    = errors.New("no authentication method configured")
	ErrTokenExpired    = errors.New("token expired")
	ErrInvalidAudience = errors.New("invalid audience")
)

type Service struct {
	apiKey       string
	jwtPublicKey *rsa.PublicKey
}

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
