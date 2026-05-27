package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddress       string `json:"listen_address"`
	StateFile           string `json:"state_file"`
	PublicURL           string `json:"public_url"`
	CookieSecure        bool   `json:"cookie_secure"`
	AllowServiceControl bool   `json:"allow_service_control"`
	SessionSecret       string `json:"session_secret"`
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "127.0.0.1:8899"
	}
	if cfg.StateFile == "" {
		cfg.StateFile = "/var/lib/turnable-manager-panel/state.json"
	}
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = "change-me"
	}
	return cfg, nil
}

func DefaultConfig() Config {
	return Config{
		ListenAddress:       "127.0.0.1:8899",
		StateFile:           "/var/lib/turnable-manager-panel/state.json",
		PublicURL:           "http://127.0.0.1:8899",
		CookieSecure:        false,
		AllowServiceControl: true,
		SessionSecret:       "change-me",
	}
}

func EnsureParent(path string, mode os.FileMode) error {
	return os.MkdirAll(filepath.Dir(path), mode)
}
