package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Settings struct {
	Tenant      string
	ClientID    string
	AccessToken string
}

var (
	tenantPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)*$`)
	tokenPattern  = regexp.MustCompile(`^[A-Za-z0-9._~+/-]+=*$`)
)

func Load(tenantFlag, clientIDFlag string) (Settings, error) {
	envPath := os.Getenv("TEAMS_ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}
	if !filepath.IsAbs(envPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return Settings{}, fmt.Errorf("find current directory: %w", err)
		}
		envPath = filepath.Join(cwd, envPath)
	}
	fileValues, err := godotenv.Read(envPath)
	if err != nil && !os.IsNotExist(err) {
		return Settings{}, fmt.Errorf("read environment file: %w", err)
	}

	tenant := firstNonEmpty(tenantFlag, os.Getenv("TEAMS_TENANT_ID"), fileValues["TEAMS_TENANT_ID"])
	clientID := firstNonEmpty(clientIDFlag, os.Getenv("TEAMS_CLIENT_ID"), fileValues["TEAMS_CLIENT_ID"])
	token := strings.TrimSpace(fileValues["ACCESS_TOKEN"])
	if token != "" && !tokenPattern.MatchString(token) {
		return Settings{}, fmt.Errorf("set ACCESS_TOKEN to the raw Graph token without the Bearer prefix")
	}
	return Settings{
		Tenant:      strings.ToLower(tenant),
		ClientID:    clientID,
		AccessToken: token,
	}, nil
}

func (s *Settings) RequireApplication() error {
	if s.Tenant == "" || s.ClientID == "" {
		return fmt.Errorf("set TEAMS_TENANT_ID and TEAMS_CLIENT_ID in .env or the environment")
	}
	if !tenantPattern.MatchString(s.Tenant) {
		return fmt.Errorf("tenant must be a directory ID or domain, not a URL")
	}
	parsedClientID, err := uuid.Parse(s.ClientID)
	if err != nil {
		return fmt.Errorf("client ID must be the Entra application UUID")
	}
	s.Tenant = strings.ToLower(s.Tenant)
	s.ClientID = parsedClientID.String()
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
