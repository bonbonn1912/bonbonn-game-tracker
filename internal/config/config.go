package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application configuration settings.
type Config struct {
	Port                     string
	Env                      string
	FaceitAPIKey             string
	FaceitBaseURL            string
	MongoURI                 string
	MongoDBName              string
	MongoPlayerCollection    string
	MongoMatchroomCollection string
	AdminUsername            string
	AdminPassword            string
	AdminHost                string
	WebhookHost              string
}

// Load reads configuration from environment variables and optional .env files.
// requireAdminPassword determines whether missing ADMIN_PASSWORD causes an error (e.g. true for server runtime).
func Load(requireAdminPassword bool) (*Config, error) {
	loadDotEnvFiles()

	port := getEnvWithFallback("PORT", "12000")
	env := getEnvWithFallback("NODE_ENV", "production")
	if env == "dev" {
		env = "development"
	}

	faceitAPIKey := getFirstEnv("FACEIT_API_KEY", "apiKey")
	faceitBaseURL := getFirstEnv("FACEIT_BASE_URL")
	if faceitBaseURL == "" {
		faceitBaseURL = "https://open.faceit.com/data/v4"
	}
	faceitBaseURL = strings.TrimRight(faceitBaseURL, "/")

	mongoURI := getFirstEnv("MONGO_URI", "MONGODB_URI", "mongoString")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDBName := getFirstEnv("MONGO_DB_NAME", "mongoDbName")
	if mongoDBName == "" {
		mongoDBName = "bonbonn"
	}

	mongoPlayerCol := getFirstEnv("MONGO_PLAYER_COLLECTION", "mongoPlayerCollectionName")
	if mongoPlayerCol == "" {
		mongoPlayerCol = "players"
	}

	mongoMatchroomCol := getFirstEnv("MONGO_MATCHROOM_COLLECTION", "mongoMatchroomCollectionName")
	if mongoMatchroomCol == "" {
		mongoMatchroomCol = "matchrooms"
	}

	adminUser := getEnvWithFallback("ADMIN_USERNAME", "admin")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if requireAdminPassword && strings.TrimSpace(adminPass) == "" {
		return nil, errors.New("ADMIN_PASSWORD environment variable is required")
	}

	adminHost := os.Getenv("ADMIN_HOST")
	webhookHost := os.Getenv("WEBHOOK_HOST")

	return &Config{
		Port:                     port,
		Env:                      env,
		FaceitAPIKey:             faceitAPIKey,
		FaceitBaseURL:            faceitBaseURL,
		MongoURI:                 mongoURI,
		MongoDBName:              mongoDBName,
		MongoPlayerCollection:    mongoPlayerCol,
		MongoMatchroomCollection: mongoMatchroomCol,
		AdminUsername:            adminUser,
		AdminPassword:            adminPass,
		AdminHost:                adminHost,
		WebhookHost:              webhookHost,
	}, nil
}

// loadDotEnvFiles checks for common .env locations and loads key=values.
func loadDotEnvFiles() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	nodeEnv := os.Getenv("NODE_ENV")
	candidates := []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, "..", ".env"),
	}

	if nodeEnv != "" {
		candidates = append([]string{
			filepath.Join(cwd, fmt.Sprintf("%s.env", nodeEnv)),
			filepath.Join(cwd, "..", fmt.Sprintf("%s.env", nodeEnv)),
		}, candidates...)
	}

	for _, path := range candidates {
		if file, err := os.Open(path); err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if _, exists := os.LookupEnv(key); !exists {
						_ = os.Setenv(key, val)
					}
				}
			}
			return
		}
	}
}

func getEnvWithFallback(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getFirstEnv(keys ...string) string {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok && val != "" {
			return val
		}
	}
	return ""
}
