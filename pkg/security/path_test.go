package security

import (
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	baseDir := "/safe/storage"

	tests := []struct {
		name    string
		path    string
		baseDir string
		wantErr error
	}{
		// Valid cases
		{
			name:    "valid relative path",
			path:    "subfolder/file.txt",
			baseDir: baseDir,
			wantErr: nil,
		},
		{
			name:    "valid simple filename",
			path:    "file.txt",
			baseDir: baseDir,
			wantErr: nil,
		},
		{
			name:    "valid path without base dir",
			path:    "file.txt",
			baseDir: "",
			wantErr: nil,
		},

		// Path traversal attacks
		{
			name:    "path traversal with ../",
			path:    "../../../etc/passwd",
			baseDir: baseDir,
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal in subdirectory",
			path:    "subdir/../../../etc/passwd",
			baseDir: baseDir,
			wantErr: ErrPathTraversal,
		},
		{
			name:    "double dot encoding",
			path:    "..%2F..%2F..%2Fetc%2Fpasswd",
			baseDir: baseDir,
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal with backslash",
			path:    "..\\..\\..\\windows\\system32",
			baseDir: baseDir,
			wantErr: ErrPathTraversal,
		},

		// Null byte injection
		{
			name:    "null byte injection",
			path:    "file.txt\x00.exe",
			baseDir: baseDir,
			wantErr: ErrUnsafeCharacters,
		},

		// Empty paths
		{
			name:    "empty path",
			path:    "",
			baseDir: baseDir,
			wantErr: ErrEmptyFilename,
		},

		// Excessive length
		{
			name:    "excessively long path",
			path:    strings.Repeat("a", MaxPathLength+1),
			baseDir: baseDir,
			wantErr: ErrInvalidFilename,
		},

		// Edge cases with cleaned paths
		{
			name:    "path with extra slashes",
			path:    "folder//subfolder///file.txt",
			baseDir: baseDir,
			wantErr: nil, // Should be cleaned and allowed
		},
		{
			name:    "current directory reference",
			path:    "./file.txt",
			baseDir: baseDir,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path, tt.baseDir)
			if (err == nil) != (tt.wantErr == nil) {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  error
	}{
		// Valid cases
		{
			name:     "simple filename",
			filename: "document.txt",
			want:     "document.txt",
			wantErr:  nil,
		},
		{
			name:     "filename with spaces",
			filename: "my document.txt",
			want:     "my_document.txt",
			wantErr:  nil,
		},

		// Unsafe characters
		{
			name:     "filename with unsafe chars",
			filename: "file<>:\"|?*.txt",
			want:     "file_.txt", // Consecutive unsafe chars collapsed to single underscore
			wantErr:  nil,
		},
		{
			name:     "filename with path separators",
			filename: "folder/file\\name.txt",
			want:     "folder_file_name.txt",
			wantErr:  nil,
		},

		// Reserved Windows names
		{
			name:     "reserved name CON",
			filename: "CON.txt",
			want:     "_CON.txt",
			wantErr:  nil,
		},
		{
			name:     "reserved name con lowercase",
			filename: "con.txt",
			want:     "_con.txt",
			wantErr:  nil,
		},
		{
			name:     "reserved name PRN",
			filename: "PRN",
			want:     "_PRN",
			wantErr:  nil,
		},

		// Edge cases
		{
			name:     "empty filename",
			filename: "",
			want:     "",
			wantErr:  ErrEmptyFilename,
		},
		{
			name:     "whitespace only",
			filename: "   ",
			want:     "",
			wantErr:  ErrEmptyFilename,
		},
		{
			name:     "dots only",
			filename: "...",
			want:     "",
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "underscores only",
			filename: "___",
			want:     "",
			wantErr:  ErrInvalidFilename,
		},

		// Length limits
		{
			name:     "long filename with extension",
			filename: strings.Repeat("a", MaxFilenameLength) + ".txt",
			want:     strings.Repeat("a", MaxFilenameLength-4) + ".txt",
			wantErr:  nil,
		},
		{
			name:     "long filename without extension",
			filename: strings.Repeat("b", MaxFilenameLength+10),
			want:     strings.Repeat("b", MaxFilenameLength),
			wantErr:  nil,
		},

		// Multiple consecutive unsafe chars
		{
			name:     "multiple consecutive spaces",
			filename: "file   name.txt",
			want:     "file_name.txt",
			wantErr:  nil,
		},
		{
			name:     "mixed unsafe characters",
			filename: "file<>|name?*.txt",
			want:     "file_name_.txt",
			wantErr:  nil,
		},

		// Leading/trailing issues
		{
			name:     "leading dots and underscores",
			filename: "...___file.txt",
			want:     "file.txt",
			wantErr:  nil,
		},
		{
			name:     "trailing dots and underscores",
			filename: "file.txt...__",
			want:     "file.txt",
			wantErr:  nil,
		},

		// Invalid UTF-8
		{
			name:     "invalid utf-8",
			filename: "file\xff\xfe.txt",
			want:     "",
			wantErr:  ErrUnsafeCharacters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeFilename(tt.filename)
			if (err == nil) != (tt.wantErr == nil) {
				t.Errorf("SanitizeFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("SanitizeFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateSecureFilename(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		extension string
		wantErr   bool
	}{
		{
			name:      "valid prefix and extension",
			prefix:    "webhook",
			extension: "json",
			wantErr:   false,
		},
		{
			name:      "no prefix with extension",
			prefix:    "",
			extension: "txt",
			wantErr:   false,
		},
		{
			name:      "prefix without extension",
			prefix:    "data",
			extension: "",
			wantErr:   false,
		},
		{
			name:      "no prefix no extension",
			prefix:    "",
			extension: "",
			wantErr:   false,
		},
		{
			name:      "unsafe prefix",
			prefix:    "webhook<>|",
			extension: "json",
			wantErr:   false, // Should sanitize the prefix
		},
		{
			name:      "extension with dot",
			prefix:    "webhook",
			extension: ".json",
			wantErr:   false,
		},
		{
			name:      "invalid extension",
			prefix:    "webhook",
			extension: "js\x00on",
			wantErr:   false, // Should sanitize the extension, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateSecureFilename(tt.prefix, tt.extension)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecureFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the generated filename is not empty
				if got == "" {
					t.Errorf("GenerateSecureFilename() returned empty filename")
				}

				// Verify the filename contains random component
				if !strings.Contains(got, "_") && tt.prefix != "" {
					t.Errorf("GenerateSecureFilename() should contain underscore separator, got %s", got)
				}

				// Verify extension is present if requested (after sanitization)
				if tt.extension != "" {
					if !strings.Contains(got, ".") {
						t.Errorf("GenerateSecureFilename() should contain extension, got %s", got)
					}
					// For the null byte test case, the extension becomes "js_on" after sanitization
					if tt.extension == "js\x00on" && !strings.HasSuffix(got, ".js_on") {
						t.Errorf("GenerateSecureFilename() should end with sanitized extension .js_on, got %s", got)
					}
				}

				// Test that generated filenames are unique
				got2, err2 := GenerateSecureFilename(tt.prefix, tt.extension)
				if err2 != nil {
					t.Errorf("Second call to GenerateSecureFilename() failed: %v", err2)
				}
				if got == got2 {
					t.Errorf("GenerateSecureFilename() should generate unique filenames, got duplicate: %s", got)
				}
			}
		})
	}
}

func TestIsSecurePath(t *testing.T) {
	baseDir := "/safe/storage"

	tests := []struct {
		name     string
		fullPath string
		baseDir  string
		wantErr  bool
	}{
		{
			name:     "valid secure path",
			fullPath: "subfolder/document.txt",
			baseDir:  baseDir,
			wantErr:  false,
		},
		{
			name:     "path traversal attack",
			fullPath: "../../../etc/passwd",
			baseDir:  baseDir,
			wantErr:  true,
		},
		{
			name:     "unsafe filename in path",
			fullPath: "subfolder/file<>.txt",
			baseDir:  baseDir,
			wantErr:  false, // SanitizeFilename doesn't fail on unsafe chars, it cleans them
		},
		{
			name:     "empty filename in path",
			fullPath: "subfolder/.",
			baseDir:  baseDir,
			wantErr:  true,
		},
		{
			name:     "null byte in path",
			fullPath: "subfolder/file\x00.txt",
			baseDir:  baseDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsSecurePath(tt.fullPath, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSecurePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkValidatePath(b *testing.B) {
	path := "subfolder/document.txt"
	baseDir := "/safe/storage"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidatePath(path, baseDir)
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	filename := "my document<>:|?*.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeFilename(filename)
	}
}

func BenchmarkGenerateSecureFilename(b *testing.B) {
	prefix := "webhook"
	extension := "json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSecureFilename(prefix, extension)
	}
}

// Test edge cases for security
func TestSecurityEdgeCases(t *testing.T) {
	t.Run("filename with null byte should fail validation", func(t *testing.T) {
		path := "file\x00.exe"
		err := ValidatePath(path, "/safe")
		if err == nil {
			t.Error("Expected error for null byte, got nil")
		}
	})

	t.Run("extremely long path should fail", func(t *testing.T) {
		longPath := strings.Repeat("a/", 3000) + "file.txt"
		err := ValidatePath(longPath, "/safe")
		if err == nil {
			t.Error("Expected error for extremely long path, got nil")
		}
	})

	t.Run("path with mixed traversal attempts", func(t *testing.T) {
		paths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"folder/../../../etc/passwd",
			"./../../etc/passwd",
		}

		for _, path := range paths {
			err := ValidatePath(path, "/safe")
			if err == nil {
				t.Errorf("Expected error for path traversal: %s", path)
			}
		}
	})
}
