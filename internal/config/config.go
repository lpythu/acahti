package config

import (
	"os"
	"strings"
)

// Version is set by ldflags; env ACAHTI_VERSION wins at runtime.
var Version = "dev"

type Config struct {
	Listen            string
	RootURL           string
	Domain            string
	GitSSHPort        string
	Org               string
	ForgejoURL        string
	WoodpeckerURL     string
	AdminToken        string
	WoodpeckerTok     string
	SessionSecret     string
	AdminUser         string
	Version           string
	ForgejoVersion    string
	WoodpeckerVersion string
	PostgresVersion   string
	DataDir           string
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Listen:            getenv("ACAHTI_LISTEN", ":8080"),
		RootURL:           strings.TrimRight(getenv("ROOT_URL", "http://127.0.0.1:8080"), "/"),
		Domain:            getenv("DOMAIN", ""),
		GitSSHPort:        getenv("GIT_SSH_PORT", "2222"),
		Org:               getenv("ACAHTI_ORG", "acme"),
		ForgejoURL:        strings.TrimRight(getenv("FORGEJO_URL", "http://forgejo:3000"), "/"),
		WoodpeckerURL:     strings.TrimRight(getenv("WOODPECKER_URL", "http://woodpecker:8000/ci"), "/"),
		AdminToken:        getenv("ACAHTI_ADMIN_TOKEN", ""),
		WoodpeckerTok:     getenv("WOODPECKER_TOKEN", ""),
		SessionSecret:     getenv("ACAHTI_SESSION_SECRET", "dev-only-change-me"),
		AdminUser:         getenv("ACAHTI_ADMIN_USER", "acahti"),
		Version:           getenv("ACAHTI_VERSION", Version),
		ForgejoVersion:    getenv("FORGEJO_VERSION", "15.0.8"),
		WoodpeckerVersion: getenv("WOODPECKER_VERSION", "3.18.1"),
		PostgresVersion:   getenv("POSTGRES_VERSION", "16-alpine"),
		DataDir:           getenv("ACAHTI_DATA", "/var/lib/acahti"),
	}
}
