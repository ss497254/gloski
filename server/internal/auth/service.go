package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ss497254/gloski/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrInvalidAPIKey   = errors.New("invalid API key")
)

type Service struct {
	passwordHash string
	jwtSecret    []byte
	tokenExpiry  time.Duration
	apiKey       string
}

type Claims struct {
	jwt.RegisteredClaims
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		passwordHash: cfg.PasswordHash,
		jwtSecret:    []byte(cfg.JWTSecret),
		tokenExpiry:  time.Duration(cfg.TokenExpiry) * time.Hour,
		apiKey:       cfg.APIKey,
	}
}

func (s *Service) Login(password string) (string, int, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password)); err != nil {
		return "", 0, ErrInvalidPassword
	}

	expiresAt := time.Now().Add(s.tokenExpiry)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, int(s.tokenExpiry.Seconds()), nil
}

func (s *Service) ValidateToken(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
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

// ValidateAPIKey checks if the provided API key is valid
func (s *Service) ValidateAPIKey(key string) error {
	if s.apiKey == "" {
		return ErrInvalidAPIKey
	}
	if key != s.apiKey {
		return ErrInvalidAPIKey
	}
	return nil
}

// HasAPIKey returns true if API key authentication is configured
func (s *Service) HasAPIKey() bool {
	return s.apiKey != ""
}

// HashPassword creates a bcrypt hash of a password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
