package runtimeconfig

import (
	"reflect"
	"strings"
	"testing"
)

func fromMap(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaultsToCLI(t *testing.T) {
	t.Parallel()
	cfg, err := load(fromMap(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultType != ExecutionTypeCLI || cfg.CloudEnabled {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	want := PublicConfig{
		DefaultType:    ExecutionTypeCLI,
		EnabledTypes:   []ExecutionType{ExecutionTypeCLI},
		CloudProviders: []string{},
	}
	if got := cfg.Public(); !reflect.DeepEqual(got, want) {
		t.Fatalf("public config = %+v, want %+v", got, want)
	}
}

func TestLoadRejectsCloudDefaultWhenDisabled(t *testing.T) {
	t.Parallel()
	_, err := load(fromMap(map[string]string{"CHIMII_RUNTIME_DEFAULT": "cloud"}))
	if err == nil || !strings.Contains(err.Error(), "requires CHIMII_CLOUD_RUNTIME_ENABLED=true") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsMalformedCloudEnabled(t *testing.T) {
	t.Parallel()
	_, err := load(fromMap(map[string]string{"CHIMII_CLOUD_RUNTIME_ENABLED": "sometimes"}))
	if err == nil || !strings.Contains(err.Error(), "must be a boolean") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCloudProviders(t *testing.T) {
	t.Parallel()
	cfg, err := load(fromMap(map[string]string{
		"CHIMII_CLOUD_RUNTIME_ENABLED":                 "true",
		"CHIMII_RUNTIME_DEFAULT":                       "cloud",
		"CHIMII_CLOUD_RUNTIME_PROVIDERS":               "openai,anthropic",
		"CHIMII_CLOUD_RUNTIME_OPENAI_API_KEY":          "openai-secret",
		"CHIMII_CLOUD_RUNTIME_OPENAI_DEFAULT_MODEL":    "openai-test-model",
		"CHIMII_CLOUD_RUNTIME_ANTHROPIC_API_KEY":       "anthropic-secret",
		"CHIMII_CLOUD_RUNTIME_ANTHROPIC_DEFAULT_MODEL": "anthropic-test-model",
		"CHIMII_CLOUD_RUNTIME_MAX_CONCURRENT_TASKS":    "4",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SupportsCloudProvider(ProviderOpenAI) || !cfg.SupportsCloudProvider(ProviderAnthropic) {
		t.Fatalf("providers not enabled: %+v", cfg.CloudProviders)
	}
	if cfg.MaxConcurrentTasks != 4 {
		t.Fatalf("max concurrent = %d", cfg.MaxConcurrentTasks)
	}
	pub := cfg.Public()
	if !reflect.DeepEqual(pub.CloudProviders, []string{ProviderAnthropic, ProviderOpenAI}) {
		t.Fatalf("public providers = %v", pub.CloudProviders)
	}
	if strings.Contains(strings.TrimSpace(strings.Join(pub.CloudProviders, ",")), "secret") {
		t.Fatal("public config leaked a secret")
	}
}

func TestLoadRejectsUnconfiguredProvider(t *testing.T) {
	t.Parallel()
	_, err := load(fromMap(map[string]string{
		"CHIMII_CLOUD_RUNTIME_ENABLED":   "true",
		"CHIMII_CLOUD_RUNTIME_PROVIDERS": "openai",
	}))
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsProviderWithoutDefaultModel(t *testing.T) {
	t.Parallel()
	_, err := load(fromMap(map[string]string{
		"CHIMII_CLOUD_RUNTIME_ENABLED":        "true",
		"CHIMII_CLOUD_RUNTIME_PROVIDERS":      "openai",
		"CHIMII_CLOUD_RUNTIME_OPENAI_API_KEY": "openai-secret",
	}))
	if err == nil || !strings.Contains(err.Error(), "OPENAI_DEFAULT_MODEL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsUnknownProvider(t *testing.T) {
	t.Parallel()
	_, err := load(fromMap(map[string]string{
		"CHIMII_CLOUD_RUNTIME_ENABLED":   "true",
		"CHIMII_CLOUD_RUNTIME_PROVIDERS": "gemini",
	}))
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
