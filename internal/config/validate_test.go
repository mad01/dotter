package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
		setupEnv map[string]string // For setting environment variables
	}{
		{
			name:  "tilde expansion",
			input: "~/testpath",
			want:  filepath.Join(homeDir, "testpath"),
		},
		{
			name:  "no tilde, no env vars",
			input: "/some/absolute/path",
			want:  "/some/absolute/path",
		},
		{
			name:     "with env var",
			input:    "$TEST_VAR/path",
			want:     "/tmp/testvalue/path",
			setupEnv: map[string]string{"TEST_VAR": "/tmp/testvalue"},
		},
		{
			name:     "tilde and env var",
			input:    "~/$TEST_VAR_SUFFIX",
			want:     filepath.Join(homeDir, "suffixpath"),
			setupEnv: map[string]string{"TEST_VAR_SUFFIX": "suffixpath"},
		},
		{
			name:  "empty path",
			input: "",
			want:  "",
		},
		{
			name:  "only tilde",
			input: "~",
			want:  homeDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for the test
			originalEnv := make(map[string]string)
			for key, value := range tt.setupEnv {
				if origVal, isset := os.LookupEnv(key); isset {
					originalEnv[key] = origVal
				}
				os.Setenv(key, value)
			}
			// Teardown: Restore original environment variables
			defer func() {
				for key := range tt.setupEnv {
					if origVal, isset := originalEnv[key]; isset {
						os.Setenv(key, origVal)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			got, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
