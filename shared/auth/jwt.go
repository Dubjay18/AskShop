package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims defines the custom JWT claims used across the codebase
// Add more fields (roles, permissions, etc.) as needed.
type Claims struct {
	UserID    string   `json:"uid"`
	FirstName string   `json:"fname"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// Config holds configurable JWT parameters loaded from env or provided by caller
// ExpiresIn / RefreshExpiresIn are durations in time.
type Config struct {
	Issuer           string
	Audience         []string
	AccessSecret     string
	RefreshSecret    string
	AccessExpiresIn  time.Duration
	RefreshExpiresIn time.Duration
}

// Manager encapsulates JWT creation & parsing
// Designed to be shared (thread-safe) across services.
type Manager struct {
	cfg Config
}

// NewManager constructs a Manager with the provided config
func NewManager(cfg Config) *Manager { return &Manager{cfg: cfg} }

// GeneratePair generates an access & refresh token pair with supplied base claims
func (m *Manager) GeneratePair(base Claims) (accessToken string, refreshToken string, err error) {
	now := time.Now().UTC()
	accessClaims := base
	accessClaims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    m.cfg.Issuer,
		Subject:   base.UserID,
		Audience:  m.cfg.Audience,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.AccessExpiresIn)),
	}
	refreshClaims := base
	refreshClaims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    m.cfg.Issuer,
		Subject:   base.UserID,
		Audience:  m.cfg.Audience,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.RefreshExpiresIn)),
		ID:        accessClaims.ID,
	}
	at, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(m.cfg.AccessSecret))
	if err != nil {
		return "", "", err
	}
	rt, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(m.cfg.RefreshSecret))
	if err != nil {
		return "", "", err
	}
	return at, rt, nil
}

// ParseAccess validates and parses an access token
func (m *Manager) ParseAccess(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, []byte(m.cfg.AccessSecret))
}

// ParseRefresh validates and parses a refresh token
func (m *Manager) ParseRefresh(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, []byte(m.cfg.RefreshSecret))
}

func (m *Manager) parse(tokenStr string, secret []byte) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
