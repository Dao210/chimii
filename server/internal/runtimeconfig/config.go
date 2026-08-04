package runtimeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ExecutionType string

const (
	ExecutionTypeCLI   ExecutionType = "cli"
	ExecutionTypeCloud ExecutionType = "cloud"

	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
)

const (
	defaultMaxConcurrentTasks = 8
	defaultTaskTimeout        = 2 * time.Hour
	defaultIdleTimeout        = 30 * time.Minute
	defaultWorkRoot           = "/var/lib/chimii/cloud-runtime"
)

var supportedCloudProviders = map[string]struct{}{
	ProviderAnthropic: {},
	ProviderOpenAI:    {},
}

type ProviderConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
}

type Config struct {
	DefaultType        ExecutionType
	CloudEnabled       bool
	CloudProviders     map[string]ProviderConfig
	MaxConcurrentTasks int
	WorkRoot           string
	TaskTimeout        time.Duration
	IdleTimeout        time.Duration
}

type PublicConfig struct {
	DefaultType    ExecutionType   `json:"default_type"`
	EnabledTypes   []ExecutionType `json:"enabled_types"`
	CloudProviders []string        `json:"cloud_providers"`
}

func LoadFromEnv() (Config, error) {
	return load(os.Getenv)
}

func MustLoadFromEnv() Config {
	cfg, err := LoadFromEnv()
	if err != nil {
		panic(fmt.Sprintf("invalid runtime configuration: %v", err))
	}
	return cfg
}

func load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DefaultType:        ExecutionType(strings.ToLower(strings.TrimSpace(getenv("CHIMII_RUNTIME_DEFAULT")))),
		CloudProviders:     make(map[string]ProviderConfig),
		MaxConcurrentTasks: defaultMaxConcurrentTasks,
		WorkRoot:           strings.TrimSpace(getenv("CHIMII_CLOUD_RUNTIME_WORK_ROOT")),
		TaskTimeout:        defaultTaskTimeout,
		IdleTimeout:        defaultIdleTimeout,
	}
	if cfg.DefaultType == "" {
		cfg.DefaultType = ExecutionTypeCLI
	}
	if cfg.DefaultType != ExecutionTypeCLI && cfg.DefaultType != ExecutionTypeCloud {
		return Config{}, fmt.Errorf("CHIMII_RUNTIME_DEFAULT must be cli or cloud, got %q", cfg.DefaultType)
	}
	var err error
	if cfg.CloudEnabled, err = boolOrDefault(getenv("CHIMII_CLOUD_RUNTIME_ENABLED"), false); err != nil {
		return Config{}, fmt.Errorf("CHIMII_CLOUD_RUNTIME_ENABLED: %w", err)
	}
	if cfg.DefaultType == ExecutionTypeCloud && !cfg.CloudEnabled {
		return Config{}, fmt.Errorf("CHIMII_RUNTIME_DEFAULT=cloud requires CHIMII_CLOUD_RUNTIME_ENABLED=true")
	}

	if cfg.MaxConcurrentTasks, err = positiveIntOrDefault(getenv("CHIMII_CLOUD_RUNTIME_MAX_CONCURRENT_TASKS"), defaultMaxConcurrentTasks); err != nil {
		return Config{}, fmt.Errorf("CHIMII_CLOUD_RUNTIME_MAX_CONCURRENT_TASKS: %w", err)
	}
	if cfg.TaskTimeout, err = durationOrDefault(getenv("CHIMII_CLOUD_RUNTIME_TASK_TIMEOUT"), defaultTaskTimeout); err != nil {
		return Config{}, fmt.Errorf("CHIMII_CLOUD_RUNTIME_TASK_TIMEOUT: %w", err)
	}
	if cfg.IdleTimeout, err = durationOrDefault(getenv("CHIMII_CLOUD_RUNTIME_IDLE_TIMEOUT"), defaultIdleTimeout); err != nil {
		return Config{}, fmt.Errorf("CHIMII_CLOUD_RUNTIME_IDLE_TIMEOUT: %w", err)
	}
	if cfg.WorkRoot == "" {
		cfg.WorkRoot = defaultWorkRoot
	}

	if !cfg.CloudEnabled {
		return cfg, nil
	}
	if !filepath.IsAbs(cfg.WorkRoot) {
		return Config{}, fmt.Errorf("CHIMII_CLOUD_RUNTIME_WORK_ROOT must be an absolute path")
	}

	providers := splitCSV(getenv("CHIMII_CLOUD_RUNTIME_PROVIDERS"))
	if len(providers) == 0 {
		providers = []string{ProviderAnthropic, ProviderOpenAI}
	}
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		provider = strings.ToLower(provider)
		if _, ok := supportedCloudProviders[provider]; !ok {
			return Config{}, fmt.Errorf("unsupported cloud runtime provider %q", provider)
		}
		if _, duplicate := seen[provider]; duplicate {
			continue
		}
		seen[provider] = struct{}{}

		prefix := "CHIMII_CLOUD_RUNTIME_" + strings.ToUpper(provider)
		providerCfg := ProviderConfig{
			APIKey:       strings.TrimSpace(getenv(prefix + "_API_KEY")),
			BaseURL:      strings.TrimSpace(getenv(prefix + "_BASE_URL")),
			DefaultModel: strings.TrimSpace(getenv(prefix + "_DEFAULT_MODEL")),
		}
		if providerCfg.APIKey == "" {
			return Config{}, fmt.Errorf("%s_API_KEY is required for enabled provider %q", prefix, provider)
		}
		if providerCfg.DefaultModel == "" {
			return Config{}, fmt.Errorf("%s_DEFAULT_MODEL is required for enabled provider %q", prefix, provider)
		}
		if providerCfg.BaseURL == "" {
			switch provider {
			case ProviderAnthropic:
				providerCfg.BaseURL = "https://api.anthropic.com"
			case ProviderOpenAI:
				providerCfg.BaseURL = "https://api.openai.com/v1"
			}
		}
		cfg.CloudProviders[provider] = providerCfg
	}
	if len(cfg.CloudProviders) == 0 {
		return Config{}, fmt.Errorf("at least one cloud runtime provider is required")
	}
	return cfg, nil
}

func (c Config) SupportsExecutionType(executionType ExecutionType) bool {
	switch executionType {
	case ExecutionTypeCLI:
		return true
	case ExecutionTypeCloud:
		return c.CloudEnabled
	default:
		return false
	}
}

func (c Config) SupportsCloudProvider(provider string) bool {
	if !c.CloudEnabled {
		return false
	}
	_, ok := c.CloudProviders[strings.ToLower(strings.TrimSpace(provider))]
	return ok
}

func (c Config) Public() PublicConfig {
	defaultType := c.DefaultType
	if defaultType == "" {
		defaultType = ExecutionTypeCLI
	}
	pub := PublicConfig{
		DefaultType:    defaultType,
		EnabledTypes:   []ExecutionType{ExecutionTypeCLI},
		CloudProviders: []string{},
	}
	if !c.CloudEnabled {
		return pub
	}
	pub.EnabledTypes = append(pub.EnabledTypes, ExecutionTypeCloud)
	pub.CloudProviders = make([]string, 0, len(c.CloudProviders))
	for provider := range c.CloudProviders {
		pub.CloudProviders = append(pub.CloudProviders, provider)
	}
	sort.Strings(pub.CloudProviders)
	return pub
}

func boolOrDefault(raw string, fallback bool) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("must be a boolean")
	}
	return value, nil
}

func positiveIntOrDefault(raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return value, nil
}

func durationOrDefault(raw string, fallback time.Duration) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("must be a positive Go duration")
	}
	return value, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
