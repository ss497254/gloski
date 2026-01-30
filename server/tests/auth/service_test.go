package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "with API key only",
			config: &config.Config{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "with invalid JWT key",
			config: &config.Config{
				APIKey:       "test-key",
				JWTPublicKey: "invalid-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := auth.NewService(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc == nil {
				t.Fatal("service is nil")
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		configKey string
		testKey   string
		wantErr   bool
	}{
		{
			name:      "valid key",
			configKey: "secret-key",
			testKey:   "secret-key",
			wantErr:   false,
		},
		{
			name:      "invalid key",
			configKey: "secret-key",
			testKey:   "wrong-key",
			wantErr:   true,
		},
		{
			name:      "empty key",
			configKey: "secret-key",
			testKey:   "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey: tt.configKey,
			}
			svc, err := auth.NewService(cfg)
			if err != nil {
				t.Fatalf("failed to create service: %v", err)
			}

			err = svc.ValidateAPIKey(tt.testKey)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	publicKeyPEM := exportRSAPublicKeyAsPEM(&privateKey.PublicKey)

	t.Run("valid token", func(t *testing.T) {
		cfg := &config.Config{
			APIKey:       "test",
			JWTPublicKey: publicKeyPEM,
		}

		svc, err := auth.NewService(cfg)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		// Create a valid token
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": "test-user",
			"exp": time.Now().Add(time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		// Validate token
		err = svc.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		cfg := &config.Config{
			APIKey:       "test",
			JWTPublicKey: publicKeyPEM,
		}

		svc, err := auth.NewService(cfg)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		// Create an expired token
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": "test-user",
			"exp": time.Now().Add(-time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		// Validate token
		err = svc.ValidateToken(tokenString)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("invalid token string", func(t *testing.T) {
		cfg := &config.Config{
			APIKey:       "test",
			JWTPublicKey: publicKeyPEM,
		}

		svc, err := auth.NewService(cfg)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		err = svc.ValidateToken("invalid.token.string")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})
}

func TestHasAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "has API key",
			apiKey: "test-key",
			want:   true,
		},
		{
			name:   "no API key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey: tt.apiKey,
			}

			// Skip if no auth method
			if tt.apiKey == "" {
				cfg.JWTPublicKey = generateTestPublicKey(t)
			}

			svc, err := auth.NewService(cfg)
			if err != nil {
				t.Fatalf("failed to create service: %v", err)
			}

			got := svc.HasAPIKey()
			if got != tt.want {
				t.Errorf("HasAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasJWT(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		want      bool
	}{
		{
			name:      "has JWT key",
			publicKey: generateTestPublicKey(t),
			want:      true,
		},
		{
			name:      "no JWT key",
			publicKey: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey:       "test",
				JWTPublicKey: tt.publicKey,
			}

			svc, err := auth.NewService(cfg)
			if err != nil {
				t.Fatalf("failed to create service: %v", err)
			}

			got := svc.HasJWT()
			if got != tt.want {
				t.Errorf("HasJWT() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to export RSA public key as PEM
func exportRSAPublicKeyAsPEM(pubkey *rsa.PublicKey) string {
	pubkeyBytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		panic(err)
	}

	pubkeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubkeyBytes,
	})

	return string(pubkeyPEM)
}

// Helper to generate test public key
func generateTestPublicKey(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return exportRSAPublicKeyAsPEM(&privateKey.PublicKey)
}
