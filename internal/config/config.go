package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const configFileName = "config.json"

var (
	ErrConfigStoreNil = errors.New("store is nil")
	ErrConfigNil      = errors.New("config is nil")
)

type Config struct {
	CurrentRoot                string `json:"current_root" mapstructure:"current_root"`
	ZapretStateRefreshInterval string `json:"zapret_state_refresh_interval" mapstructure:"zapret_state_refresh_interval"`
	ZapretReleaseCheckInterval string `json:"zapret_release_check_interval" mapstructure:"zapret_release_check_interval"`
	ZapretReleaseRetryInterval string `json:"zapret_release_retry_interval" mapstructure:"zapret_release_retry_interval"`
	ElevatedTaskName           string `json:"elevated_task_name" mapstructure:"elevated_task_name"`
	Language                   string `json:"language" mapstructure:"language"`

	CustomStrategies []string `json:"custom_strategies" mapstructure:"custom_strategies"`
	UserLocalRoots   []string `json:"user_local_roots" mapstructure:"user_local_roots"`

	GlobalSettingsEnabled bool   `json:"global_settings_enabled" mapstructure:"global_settings_enabled"`
	GlobalGameFilter      string `json:"global_game_filter" mapstructure:"global_game_filter"`
	GlobalIPSetMode       string `json:"global_ipset_mode" mapstructure:"global_ipset_mode"`

	// VPNManageEnabled: when on, zapret is stopped when a VPN (tun adapter)
	// connects and restored when it disconnects — but only if we were the ones
	// who stopped it.
	VPNManageEnabled bool `json:"vpn_manage_enabled" mapstructure:"vpn_manage_enabled"`

	ZapretAutoRunEnabled              bool `json:"zapret_auto_run_enabled" mapstructure:"zapret_auto_run_enabled"`
	ZapretStateRefreshEnabled         bool `json:"zapret_state_refresh_enabled" mapstructure:"zapret_state_refresh_enabled"`
	ZapretReleaseCheckEnabled         bool `json:"zapret_release_check_enabled" mapstructure:"zapret_release_check_enabled"`
	ZapretReleaseRetryIfFailedEnabled bool `json:"zapret_release_retry_if_failed_enabled" mapstructure:"zapret_release_retry_if_failed_enabled"`
}

type Store struct {
	v    *viper.Viper
	path string
}

func NewConfigStore(path string) *Store {
	if path == "" {
		path = DefaultPath()
	}
	return &Store{
		path: path,
		v:    newViper(path),
	}
}

func Load(path string) (*Store, *Config, error) {
	store := NewConfigStore(path)

	cfg, err := store.Read()
	if err != nil {
		return nil, nil, err
	}

	return store, cfg, nil
}

func (s *Store) Read() (*Config, error) {
	if s == nil {
		return nil, ErrConfigStoreNil
	}

	cfg := Default()
	if err := s.v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		return cfg, nil
	}

	if err := s.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return cfg, nil
}

func (s *Store) Write(cfg *Config) error {
	if s == nil {
		return ErrConfigStoreNil
	}

	if cfg == nil {
		return ErrConfigNil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	s.v.Set("current_root", cfg.CurrentRoot)
	s.v.Set("zapret_auto_run_enabled", cfg.ZapretAutoRunEnabled)
	s.v.Set("zapret_state_refresh_enabled", cfg.ZapretStateRefreshEnabled)
	s.v.Set("zapret_state_refresh_interval", cfg.ZapretStateRefreshInterval)
	s.v.Set("zapret_release_check_enabled", cfg.ZapretReleaseCheckEnabled)
	s.v.Set("zapret_release_check_interval", cfg.ZapretReleaseCheckInterval)
	s.v.Set("zapret_release_retry_if_failed_enabled", cfg.ZapretReleaseRetryIfFailedEnabled)
	s.v.Set("zapret_release_retry_interval", cfg.ZapretReleaseRetryInterval)
	s.v.Set("elevated_task_name", cfg.ElevatedTaskName)
	s.v.Set("language", cfg.Language)
	s.v.Set("custom_strategies", cfg.CustomStrategies)
	s.v.Set("user_local_roots", cfg.UserLocalRoots)
	s.v.Set("global_settings_enabled", cfg.GlobalSettingsEnabled)
	s.v.Set("global_game_filter", cfg.GlobalGameFilter)
	s.v.Set("global_ipset_mode", cfg.GlobalIPSetMode)
	s.v.Set("vpn_manage_enabled", cfg.VPNManageEnabled)

	err := s.v.WriteConfigAs(s.path)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func (c *Config) StateRefreshIntervalDuration() (time.Duration, error) {
	if !c.ZapretStateRefreshEnabled {
		return 0, nil
	}
	return ParseInterval(c.ZapretStateRefreshInterval)
}

func (c *Config) ReleaseCheckIntervalDuration() (time.Duration, error) {
	if !c.ZapretReleaseCheckEnabled {
		return 0, nil
	}
	return ParseInterval(c.ZapretReleaseCheckInterval)
}

func (c *Config) ReleaseRetryIntervalDuration() (time.Duration, error) {
	if !c.ZapretReleaseRetryIfFailedEnabled {
		return 0, nil
	}
	return ParseInterval(c.ZapretReleaseRetryInterval)
}

func newViper(path string) *viper.Viper {
	v := viper.New()
	v.SetConfigFile(path)

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext != "" {
		v.SetConfigType(ext)
	}

	return v
}

func Default() *Config {
	return &Config{
		CurrentRoot:                       "",
		ZapretAutoRunEnabled:              true,
		ZapretStateRefreshEnabled:         true,
		ZapretStateRefreshInterval:        "30s",
		ZapretReleaseCheckEnabled:         true,
		ZapretReleaseCheckInterval:        "1h",
		ZapretReleaseRetryIfFailedEnabled: true,
		ZapretReleaseRetryInterval:        "10m",
		ElevatedTaskName:                  "ZapretTrayManager",
	}
}

func DefaultPath() string {
	dir := ExecutableDir()
	if dir != "" {
		return filepath.Join(dir, configFileName)
	}

	return configFileName
}

func ExecutableDir() string {
	executable, err := os.Executable()
	if err == nil && executable != "" {
		return filepath.Dir(executable)
	}

	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}

	return ""
}

func ParseInterval(interval string) (time.Duration, error) {
	d, err := time.ParseDuration(interval)
	if err != nil {
		return 0, fmt.Errorf("invalid interval format: %w", err)
	}
	return d, nil
}
