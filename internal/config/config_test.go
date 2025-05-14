package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `rateLimits:
  - apiKey: TEST_KEY
    endpoints:
      - path: /test
        rpm: 100
        tpm: 10`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test cases
	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(*testing.T, *Configuration)
	}{
		{
			name:    "Valid config",
			path:    tmpfile.Name(),
			wantErr: false,
			check: func(t *testing.T, cfg *Configuration) {
				if len(cfg.RateLimits) != 1 {
					t.Error("Expected 1 rate limit configuration")
				}
				if cfg.RateLimits[0].APIKey != "TEST_KEY" {
					t.Errorf("Expected API key TEST_KEY, got %s", cfg.RateLimits[0].APIKey)
				}
				if len(cfg.RateLimits[0].Endpoints) != 1 {
					t.Error("Expected 1 endpoint configuration")
				}
				endpoint := cfg.RateLimits[0].Endpoints[0]
				if endpoint.Path != "/test" {
					t.Errorf("Expected path /test, got %s", endpoint.Path)
				}
				if endpoint.RPM != 100 {
					t.Errorf("Expected RPM 100, got %d", endpoint.RPM)
				}
				if endpoint.TPM != 10 {
					t.Errorf("Expected TPM 10, got %d", endpoint.TPM)
				}
			},
		},
		{
			name:    "Non-existent file",
			path:    "nonexistent.yaml",
			wantErr: true,
			check:   nil,
		},
		{
			name: "Invalid YAML",
			path: func() string {
				f, _ := os.CreateTemp("", "invalid-*.yaml")
				f.Write([]byte("invalid: [yaml: content"))
				name := f.Name()
				f.Close()
				return name
			}(),
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
