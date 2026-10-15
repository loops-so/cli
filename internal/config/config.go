package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	DefaultEndpointURL = "https://app.loops.so/api/v1"
	keyringService     = "loops"
)

type Config struct {
	APIKey      string
	EndpointURL string
	Debug       bool
}

func EndpointURL() string {
	if v := os.Getenv("LOOPS_ENDPOINT_URL"); v != "" {
		return v
	}
	return DefaultEndpointURL
}

func Load(teamOverride string) (*Config, error) {
	cfg := &Config{
		EndpointURL: EndpointURL(),
	}

	if v := os.Getenv("LOOPS_API_KEY"); v != "" {
		cfg.APIKey = v
		return cfg, nil
	}

	team := teamOverride
	if team == "" {
		pc, err := LoadPersistentConfig()
		if err != nil {
			return nil, err
		}
		team = pc.ActiveTeam

		// if there is only one key, use it even if its not active
		if team == "" && len(pc.Teams) == 1 {
			team = pc.Teams[0]
		}
	}

	if team != "" {
		key, err := keyring.Get(keyringService, "key:"+team)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("could not read keyring: %w", err)
		}
		cfg.APIKey = key
	}

	if cfg.APIKey == "" {
		return nil, errors.New("no active team set — run `loops auth login <name>` to authenticate")
	}

	return cfg, nil
}
