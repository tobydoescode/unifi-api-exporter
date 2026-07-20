package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// maxScrapeTimeout caps how long a single poll may take regardless of interval.
const maxScrapeTimeout = 30 * time.Second

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
		ListenAddr: getenv("LISTEN_ADDR", ":8080"),
	}
	if c.URL == "" || c.User == "" || c.Pass == "" {
		return Config{}, fmt.Errorf("UNIFI_URL, UNIFI_USER and UNIFI_PASS are required")
	}
	insecure, err := strconv.ParseBool(getenv("UNIFI_INSECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid UNIFI_INSECURE: %w", err)
	}
	c.Insecure = insecure
	d, err := time.ParseDuration(getenv("POLL_INTERVAL", "30s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
	}
	if d <= 0 {
		return Config{}, fmt.Errorf("POLL_INTERVAL must be positive, got %s", d)
	}
	c.PollInterval = d
	return c, nil
}

// ScrapeTimeout bounds one poll: the poll interval, capped at 30s, so a slow
// scrape can never overlap the next tick.
func (c Config) ScrapeTimeout() time.Duration {
	if c.PollInterval < maxScrapeTimeout {
		return c.PollInterval
	}
	return maxScrapeTimeout
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
