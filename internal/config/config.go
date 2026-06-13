package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Schedule struct {
	Days      []time.Weekday `yaml:"days"`
	StartTime string         `yaml:"start_time"`
	Duration  time.Duration  `yaml:"duration"`
}

type Profile struct {
	Name         string        `yaml:"name"`
	Interval     time.Duration `yaml:"interval"`
	Duration     time.Duration `yaml:"duration"`
	Recording    string        `yaml:"recording"`
	MovementType string        `yaml:"movement_type"`
	Schedules    []Schedule    `yaml:"schedules"`
}

type AppConfig struct {
	DefaultProfile string    `yaml:"default_profile"`
	Profiles       []Profile `yaml:"profiles"`
}

func Load() (*AppConfig, error) {
	configDir := Dir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := &AppConfig{
				DefaultProfile: "default",
				Profiles:       []Profile{DefaultProfile()},
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	data, migrated := migrateKeys(data)

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		cfg.Profiles = []Profile{DefaultProfile()}
		cfg.DefaultProfile = "default"
	}

	if migrated {
		err := Save(&cfg)
		if err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

func Save(cfg *AppConfig) error {
	configDir := Dir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	return os.WriteFile(configPath, data, 0o644)
}

func (c *AppConfig) GetProfile(name string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile %q not found", name)
}

func (c *AppConfig) GetDefaultProfile() (*Profile, error) {
	return c.GetProfile(c.DefaultProfile)
}

func (c *AppConfig) ListProfiles() []string {
	names := make([]string, len(c.Profiles))
	for i, p := range c.Profiles {
		names[i] = p.Name
	}
	return names
}

func (c *AppConfig) DeleteProfile(name string) error {
	if name == c.DefaultProfile {
		return fmt.Errorf("cannot delete the default profile")
	}
	for i, p := range c.Profiles {
		if p.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}

func (c *AppConfig) AddProfile(p Profile) error {
	for _, existing := range c.Profiles {
		if existing.Name == p.Name {
			return fmt.Errorf("profile %q already exists", p.Name)
		}
	}
	c.Profiles = append(c.Profiles, p)
	return nil
}

func (c *AppConfig) SetDefault(name string) error {
	for _, p := range c.Profiles {
		if p.Name == name {
			c.DefaultProfile = name
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}

func migrateKeys(data []byte) ([]byte, bool) {
	s := string(data)
	migrated := false
	if strings.Contains(s, "starttime:") {
		s = strings.ReplaceAll(s, "starttime:", "start_time:")
		migrated = true
	}
	if strings.Contains(s, "movementtype:") {
		s = strings.ReplaceAll(s, "movementtype:", "movement_type:")
		migrated = true
	}
	return []byte(s), migrated
}
