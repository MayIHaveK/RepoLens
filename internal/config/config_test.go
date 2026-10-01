package config

import (
	"encoding/json"
	"github.com/MayIHaveK/RepoLens/internal/model"
	"testing"
)

func TestNormalizeFillsDefaults(t *testing.T) {
	var cfg Config
	cfg.Normalize()
	if cfg.Ref != "HEAD" {
		t.Fatalf("expected HEAD, got %q", cfg.Ref)
	}
	if cfg.Parallelism < 1 {
		t.Fatal("parallelism should be positive")
	}
	if cfg.Weights.Workload == 0 {
		t.Fatal("weights were not initialized")
	}
}

func TestPartialJSONPreservesDefaultsAndExplicitFalse(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{"enableOwnership":false,"categoryWeights":{"source":0}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	prepared, err := cfg.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	if prepared.EnableOwnership || prepared.EnableRetention || !prepared.DetectRenames || prepared.CategoryWeights[model.CategorySource] != 0 || prepared.CategoryWeights[model.CategoryTest] != 0.9 {
		t.Fatalf("incorrect overlay: %+v", prepared)
	}
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &cfg); err == nil {
		t.Fatal("unknown config accepted")
	}
}

func TestInvalidConfigurationRejected(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Weights.Workload = -0.1 },
		func(c *Config) { c.Weights = model.Weights{} },
		func(c *Config) { c.CategoryWeights[model.CategorySource] = 2 },
		func(c *Config) { c.Since = "2026-03-03"; c.Until = "2026-03-01" },
		func(c *Config) { c.Since = "not-a-date" },
		func(c *Config) { c.Parallelism = 33 },
		func(c *Config) { c.MaxInventoryFiles = -1 },
	} {
		cfg := Default()
		mutate(&cfg)
		if _, err := cfg.Prepared(); err == nil {
			t.Fatalf("invalid config accepted: %+v", cfg)
		}
	}
}

func TestPreparedIsIdempotentAndDoesNotMutateAliases(t *testing.T) {
	for _, profile := range []string{"fast", "balanced", "thorough"} {
		cfg := Default()
		cfg.Profile = profile
		cfg.Aliases[" OLD@EXAMPLE.TEST "] = "New@Example.Test"
		first, err := cfg.Prepared()
		if err != nil {
			t.Fatal(err)
		}
		second, err := first.Prepared()
		if err != nil {
			t.Fatal(err)
		}
		if first.Fingerprint() != second.Fingerprint() {
			t.Fatal("prepared config is not idempotent")
		}
		if first.Aliases["old@example.test"] != "new@example.test" || cfg.Aliases[" OLD@EXAMPLE.TEST "] != "New@Example.Test" {
			t.Fatal("alias normalization mutated caller")
		}
	}
}

func TestFingerprintChangesWithConfiguration(t *testing.T) {
	a := Default()
	b := Default()
	b.Ref = "main"
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatal("fingerprint should change when configuration changes")
	}
}
