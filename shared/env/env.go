/*
Package env provides a simple way to get environment variables.
*/
package env

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var loadOnce sync.Once

// init attempts to load a .env file once. Failures are ignored to allow
// environment variables provided by the runtime (Docker/K8s) to take precedence.
func init() {
	loadOnce.Do(func() {
		// Try current working directory first
		if err := godotenv.Load(); err == nil {
			return
		}
		// Walk up to 5 parent directories to find a .env near repo root
		if wd, err := os.Getwd(); err == nil {
			for i := 0; i < 5; i++ { // safety limit
				candidate := filepath.Join(wd, ".env")
				if _, statErr := os.Stat(candidate); statErr == nil {
					if loadErr := godotenv.Load(candidate); loadErr == nil {
						return
					}
				}
				parent := filepath.Dir(wd)
				if parent == wd { // reached filesystem root
					break
				}
				wd = parent
			}
		}
		// Optional explicit path via ENV_FILE
		if custom := os.Getenv("../.env"); custom != "" {
			if err := godotenv.Load(custom); err != nil {
				log.Printf("env: failed to load custom ENV_FILE %s: %v", custom, err)
			}
		}
	})
}

func GetString(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return val
}

func GetInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}

	return valAsInt
}

func GetBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}

	return boolVal
}
