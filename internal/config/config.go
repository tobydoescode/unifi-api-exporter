package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	URL          string
	User         string
	Pass         string
	Site         string
	Insecure     bool
	PollInterval time.Duration
	ListenAddr   string
}

// Load reads configuration from environment variables, applying defaults.
func Load() (Config, error) {
	c := Config{
		URL:        os.Getenv("UNIFI_URL"),
		User:       os.Getenv("UNIFI_USER"),
		Pass:       os.Getenv("UNIFI_PASS"),
		Site:       getenv("UNIFI_SITE", "default"),
		Insecure:   getenv("UNIFI_INSECURE", "true") == "true",
		ListenAddr: getenv("LISTEN_ADDR", ":8080"),
	}
	if c.URL == "" || c.User == "" || c.Pass == "" {
		return Config{}, fmt.Errorf("UNIFI_URL, UNIFI_USER and UNIFI_PASS are required")
	}
	d, err := time.ParseDuration(getenv("POLL_INTERVAL", "30s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
	}
	c.PollInterval = d
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
