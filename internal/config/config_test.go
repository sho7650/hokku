package config

import (
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	// Clear environment variables first
	envVars := []string{
		"HOKKU_STORAGE_PATH", "HOKKU_MAX_FILE_SIZE", "HOKKU_PORT",
		"HOKKU_AUTH_TOKEN", "HOKKU_ENV", "HOKKU_ALLOWED_EXTENSIONS",
		"HOKKU_MAX_TITLE_LENGTH", "HOKKU_MAX_DESC_LENGTH", "HOKKU_MAX_DATA_SIZE",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	t.Run("default configuration", func(t *testing.T) {
		cfg := New()

		if cfg.GetStoragePath() != DefaultStoragePath {
			t.Errorf("GetStoragePath() = %s, want %s", cfg.GetStoragePath(), DefaultStoragePath)
		}

		if cfg.GetMaxFileSize() != DefaultMaxFileSize {
			t.Errorf("GetMaxFileSize() = %d, want %d", cfg.GetMaxFileSize(), DefaultMaxFileSize)
		}

		if cfg.GetPort() != DefaultPort {
			t.Errorf("GetPort() = %d, want %d", cfg.GetPort(), DefaultPort)
		}

		if cfg.GetAuthToken() != "" {
			t.Errorf("GetAuthToken() = %s, want empty", cfg.GetAuthToken())
		}

		if cfg.IsProduction() {
			t.Error("IsProduction() should be false for development environment")
		}

		if len(cfg.GetAllowedExtensions()) == 0 {
			t.Error("GetAllowedExtensions() should return default extensions")
		}
	})

	t.Run("environment variable overrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("HOKKU_STORAGE_PATH", "/custom/path")
		os.Setenv("HOKKU_MAX_FILE_SIZE", "2048")
		os.Setenv("HOKKU_PORT", "9090")
		os.Setenv("HOKKU_AUTH_TOKEN", "secret123")
		os.Setenv("HOKKU_ENV", "production")
		os.Setenv("HOKKU_ALLOWED_EXTENSIONS", "json,xml,txt")
		os.Setenv("HOKKU_MAX_TITLE_LENGTH", "128")
		os.Setenv("HOKKU_MAX_DESC_LENGTH", "1024")
		os.Setenv("HOKKU_MAX_DATA_SIZE", "10485760")

		defer func() {
			for _, envVar := range envVars {
				os.Unsetenv(envVar)
			}
		}()

		cfg := New()

		if cfg.GetStoragePath() != "/custom/path" {
			t.Errorf("GetStoragePath() = %s, want /custom/path", cfg.GetStoragePath())
		}

		if cfg.GetMaxFileSize() != 2048 {
			t.Errorf("GetMaxFileSize() = %d, want 2048", cfg.GetMaxFileSize())
		}

		if cfg.GetPort() != 9090 {
			t.Errorf("GetPort() = %d, want 9090", cfg.GetPort())
		}

		if cfg.GetAuthToken() != "secret123" {
			t.Errorf("GetAuthToken() = %s, want secret123", cfg.GetAuthToken())
		}

		if !cfg.IsProduction() {
			t.Error("IsProduction() should be true for production environment")
		}

		expectedExts := []string{"json", "xml", "txt"}
		actualExts := cfg.GetAllowedExtensions()
		if len(actualExts) != len(expectedExts) {
			t.Errorf("GetAllowedExtensions() length = %d, want %d", len(actualExts), len(expectedExts))
		}

		if cfg.GetMaxTitleLength() != 128 {
			t.Errorf("GetMaxTitleLength() = %d, want 128", cfg.GetMaxTitleLength())
		}

		if cfg.GetMaxDescLength() != 1024 {
			t.Errorf("GetMaxDescLength() = %d, want 1024", cfg.GetMaxDescLength())
		}

		if cfg.GetMaxDataSize() != 10485760 {
			t.Errorf("GetMaxDataSize() = %d, want 10485760", cfg.GetMaxDataSize())
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json", "txt"},
				MaxTitleLength:    64,
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "empty storage path",
			config: &Config{
				StoragePath: "",
				MaxFileSize: 1024 * 1024,
				Port:        8080,
				Environment: "development",
			},
			wantErr: true,
			errMsg:  "storage path cannot be empty",
		},
		{
			name: "negative max file size",
			config: &Config{
				StoragePath: "/valid/path",
				MaxFileSize: -1,
				Port:        8080,
				Environment: "development",
			},
			wantErr: true,
			errMsg:  "max file size must be positive",
		},
		{
			name: "max file size too large",
			config: &Config{
				StoragePath: "/valid/path",
				MaxFileSize: 200 * 1024 * 1024, // 200MB > 100MB limit
				Port:        8080,
				Environment: "development",
			},
			wantErr: true,
			errMsg:  "max file size too large",
		},
		{
			name: "invalid port number - too low",
			config: &Config{
				StoragePath: "/valid/path",
				MaxFileSize: 1024 * 1024,
				Port:        0,
				Environment: "development",
			},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name: "invalid port number - too high",
			config: &Config{
				StoragePath: "/valid/path",
				MaxFileSize: 1024 * 1024,
				Port:        70000,
				Environment: "development",
			},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name: "production without auth token",
			config: &Config{
				StoragePath: "/valid/path",
				MaxFileSize: 1024 * 1024,
				Port:        8080,
				Environment: "production",
				AuthToken:   "",
			},
			wantErr: true,
			errMsg:  "auth token is required in production",
		},
		{
			name: "production with auth token",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "production",
				AuthToken:         "secure-token",
				AllowedExtensions: []string{"json"},
				MaxTitleLength:    64,
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "invalid max title length - zero",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json"},
				MaxTitleLength:    0,
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "invalid max title length",
		},
		{
			name: "invalid max title length - too large",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json"},
				MaxTitleLength:    2000, // > 1024 limit
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "invalid max title length",
		},
		{
			name: "invalid max desc length - too large",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json"},
				MaxTitleLength:    64,
				MaxDescLength:     5000, // > 4096 limit
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "invalid max description length",
		},
		{
			name: "invalid max data size - zero",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json"},
				MaxTitleLength:    64,
				MaxDescLength:     512,
				MaxDataSize:       0,
			},
			wantErr: true,
			errMsg:  "max data size must be positive",
		},
		{
			name: "empty extension in allowed list",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json", "", "txt"}, // Empty extension
				MaxTitleLength:    64,
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "empty extension in allowed extensions list",
		},
		{
			name: "extension with dot",
			config: &Config{
				StoragePath:       "/valid/path",
				MaxFileSize:       1024 * 1024,
				Port:              8080,
				Environment:       "development",
				AllowedExtensions: []string{"json", ".txt"}, // Extension with dot
				MaxTitleLength:    64,
				MaxDescLength:     512,
				MaxDataSize:       5 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "extension should not contain dot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"production lowercase", "production", true},
		{"production uppercase", "PRODUCTION", true},
		{"production mixed case", "Production", true},
		{"development", "development", false},
		{"dev", "dev", false},
		{"test", "test", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	t.Run("getEnvString", func(t *testing.T) {
		os.Setenv("TEST_STRING", "test_value")
		defer os.Unsetenv("TEST_STRING")

		result := getEnvString("TEST_STRING", "default")
		if result != "test_value" {
			t.Errorf("getEnvString() = %s, want test_value", result)
		}

		result = getEnvString("NON_EXISTENT", "default")
		if result != "default" {
			t.Errorf("getEnvString() = %s, want default", result)
		}
	})

	t.Run("getEnvInt", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 10)
		if result != 42 {
			t.Errorf("getEnvInt() = %d, want 42", result)
		}

		result = getEnvInt("NON_EXISTENT", 10)
		if result != 10 {
			t.Errorf("getEnvInt() = %d, want 10", result)
		}

		// Test invalid int
		os.Setenv("INVALID_INT", "not_a_number")
		result = getEnvInt("INVALID_INT", 10)
		if result != 10 {
			t.Errorf("getEnvInt() should return default for invalid int, got %d", result)
		}
		os.Unsetenv("INVALID_INT")
	})

	t.Run("getEnvInt64", func(t *testing.T) {
		os.Setenv("TEST_INT64", "1234567890")
		defer os.Unsetenv("TEST_INT64")

		result := getEnvInt64("TEST_INT64", 100)
		if result != 1234567890 {
			t.Errorf("getEnvInt64() = %d, want 1234567890", result)
		}

		result = getEnvInt64("NON_EXISTENT", 100)
		if result != 100 {
			t.Errorf("getEnvInt64() = %d, want 100", result)
		}
	})

	t.Run("getEnvStringSlice", func(t *testing.T) {
		os.Setenv("TEST_SLICE", "json,xml,txt")
		defer os.Unsetenv("TEST_SLICE")

		result := getEnvStringSlice("TEST_SLICE", []string{"default"})
		expected := []string{"json", "xml", "txt"}
		if len(result) != len(expected) {
			t.Errorf("getEnvStringSlice() length = %d, want %d", len(result), len(expected))
		}

		for i, v := range expected {
			if result[i] != v {
				t.Errorf("getEnvStringSlice()[%d] = %s, want %s", i, result[i], v)
			}
		}

		result = getEnvStringSlice("NON_EXISTENT", []string{"default"})
		if len(result) != 1 || result[0] != "default" {
			t.Errorf("getEnvStringSlice() = %v, want [default]", result)
		}

		// Test with spaces and empty values
		os.Setenv("TEST_SLICE_SPACES", "json, xml, , txt, ")
		result = getEnvStringSlice("TEST_SLICE_SPACES", []string{"default"})
		expected = []string{"json", "xml", "txt"}
		if len(result) != len(expected) {
			t.Errorf("getEnvStringSlice() with spaces length = %d, want %d", len(result), len(expected))
		}
		os.Unsetenv("TEST_SLICE_SPACES")
	})
}
