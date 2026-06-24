package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v2"
)

type PersistentConfig struct {
	ActiveTeam string   `yaml:"activeTeam"`
	Teams      []string `yaml:"teams"`
}

type KeyEntry struct {
	Name   string
	APIKey string
}

func configFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config directory: %w", err)
	}
	return filepath.Join(dir, "config.yml"), nil
}

func LoadPersistentConfig() (*PersistentConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &PersistentConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	var pc PersistentConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}
	return &pc, nil
}

func SavePersistentConfig(pc *PersistentConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}
	return os.Rename(tmp, path)
}

func Save(apiKey, name string) error {
	if name == "" {
		return errors.New("a name is required — use --name to give this key a name")
	}
	if err := keyring.Set(keyringService, "key:"+name, apiKey); err != nil {
		return fmt.Errorf("could not save to keyring: %w", err)
	}
	pc, err := LoadPersistentConfig()
	if err != nil {
		return err
	}
	if !slices.Contains(pc.Teams, name) {
		pc.Teams = append(pc.Teams, name)
	}
	pc.ActiveTeam = name
	return SavePersistentConfig(pc)
}

func SetActiveTeam(name string) error {
	pc, err := LoadPersistentConfig()
	if err != nil {
		return err
	}
	if name != "" && !slices.Contains(pc.Teams, name) {
		return fmt.Errorf("no key named %q — run `loops auth list` to see available keys", name)
	}
	pc.ActiveTeam = name
	return SavePersistentConfig(pc)
}

func ListKeys() ([]KeyEntry, error) {
	pc, err := LoadPersistentConfig()
	if err != nil {
		return nil, err
	}
	entries := make([]KeyEntry, 0, len(pc.Teams))
	for _, name := range pc.Teams {
		key, err := keyring.Get(keyringService, "key:"+name)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("could not read key %q: %w", name, err)
		}
		entries = append(entries, KeyEntry{Name: name, APIKey: key})
	}
	return entries, nil
}

func Get(name string) (string, error) {
	if name == "" {
		return "", errors.New("a name is required")
	}
	pc, err := LoadPersistentConfig()
	if err != nil {
		return "", err
	}
	if !slices.Contains(pc.Teams, name) {
		return "", fmt.Errorf("no key named %q — run `loops auth list` to see available keys", name)
	}
	key, err := keyring.Get(keyringService, "key:"+name)
	if err != nil {
		return "", fmt.Errorf("could not read key %q: %w", name, err)
	}
	return key, nil
}

func Delete(name string) error {
	if name == "" {
		return errors.New("a name is required — use --name to specify which key to remove")
	}
	if err := keyring.Delete(keyringService, "key:"+name); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("could not delete key %q from keyring: %w", name, err)
	}
	pc, err := LoadPersistentConfig()
	if err != nil {
		return err
	}
	pc.Teams = slices.DeleteFunc(pc.Teams, func(t string) bool { return t == name })
	if pc.ActiveTeam == name {
		pc.ActiveTeam = ""
	}
	return SavePersistentConfig(pc)
}
