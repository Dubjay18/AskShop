package auth

import (
	"askshop/shared/env"
	"time"
)

// LoadConfig builds a Config from environment variables with fallbacks.
// JWT_ACCESS_EXPIRES (minutes), JWT_REFRESH_EXPIRES (minutes)
func LoadConfig() Config {
	issuer := env.GetString("JWT_ISSUER", "askshop")
	accessSecret := env.GetString("JWT_ACCESS_SECRET", "dev_access_secret_change")
	refreshSecret := env.GetString("JWT_REFRESH_SECRET", "dev_refresh_secret_change")
	accessMins := env.GetInt("JWT_ACCESS_EXPIRES", 15)
	refreshMins := env.GetInt("JWT_REFRESH_EXPIRES", 60*24*7) // one week default
	return Config{
		Issuer:           issuer,
		Audience:         []string{"askshop-clients"},
		AccessSecret:     accessSecret,
		RefreshSecret:    refreshSecret,
		AccessExpiresIn:  time.Duration(accessMins) * time.Minute,
		RefreshExpiresIn: time.Duration(refreshMins) * time.Minute,
	}
}
