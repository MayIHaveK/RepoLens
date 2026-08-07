package config

import "testing"

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

func TestFingerprintChangesWithConfiguration(t *testing.T) {
	a := Default()
	b := Default()
	b.Ref = "main"
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatal("fingerprint should change when configuration changes")
	}
}
