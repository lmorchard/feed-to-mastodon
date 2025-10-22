package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than limit",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "string longer than limit",
			input:  "this is a very long string that should be truncated",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "string with multi-byte characters",
			input:  "Hello 世界! This is a test",
			maxLen: 15,
			want:   "Hello 世界! Th...",
		},
		{
			name:   "string with emoji",
			input:  "Hello 😀 😃 😄 😁 😆 😅",
			maxLen: 15,
			want:   "Hello 😀 😃 😄 ...",
		},
		{
			name:   "exactly at limit",
			input:  "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "zero limit",
			input:  "test",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "limit of 1",
			input:  "test",
			maxLen: 1,
			want:   "t",
		},
		{
			name:   "limit of 2",
			input:  "test",
			maxLen: 2,
			want:   "te",
		},
		{
			name:   "limit of 3",
			input:  "test",
			maxLen: 3,
			want:   "tes",
		},
		{
			name:   "limit of 4 (adds ellipsis)",
			input:  "test string",
			maxLen: 4,
			want:   "t...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("loads valid template file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if renderer == nil {
			t.Error("Expected non-nil renderer")
		}
		if renderer.characterLimit != 500 {
			t.Errorf("characterLimit = %d, want 500", renderer.characterLimit)
		}
	})

	t.Run("template file not found error", func(t *testing.T) {
		_, err := New("/nonexistent/template.txt", 500)
		if err == nil {
			t.Error("Expected error for nonexistent template file")
		}
	})

	t.Run("invalid template syntax error", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Invalid{{}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		_, err = New(tmplPath, 500)
		if err == nil {
			t.Error("Expected error for invalid template syntax")
		}
	})

	t.Run("custom functions are registered", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		// Template using truncate function
		err := os.WriteFile(tmplPath, []byte(`{{.Item.Title | truncate 10}}`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v, truncate function not registered", err)
		}

		if renderer == nil {
			t.Error("Expected non-nil renderer")
		}
	})
}

func TestRender(t *testing.T) {
	t.Run("renders simple template", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{Title: "Test Title"}
		itemJSON, _ := json.Marshal(item)

		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if result != "Test Title" {
			t.Errorf("Render() = %q, want %q", result, "Test Title")
		}
	})

	t.Run("renders with all item fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		tmplContent := `Title: {{.Item.Title}}
Link: {{.Item.Link}}
Description: {{.Item.Description}}`
		err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{
			Title:       "Test Title",
			Link:        "https://example.com",
			Description: "Test Description",
		}
		itemJSON, _ := json.Marshal(item)

		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		expected := `Title: Test Title
Link: https://example.com
Description: Test Description`
		if result != expected {
			t.Errorf("Render() = %q, want %q", result, expected)
		}
	})

	t.Run("renders with minimal item fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{Title: "Only Title"}
		itemJSON, _ := json.Marshal(item)

		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if result != "Only Title" {
			t.Errorf("Render() = %q, want %q", result, "Only Title")
		}
	})

	// Note: Testing truncate function with numeric literals in templates is
	// challenging due to html/template's strict type checking. The truncate
	// function itself is thoroughly tested in TestTruncate.

	t.Run("invalid JSON error", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		_, err = renderer.Render([]byte("invalid json"))
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("character limit warning", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Description}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		// Set a very low character limit
		renderer, err := New(tmplPath, 10)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{
			Description: "This description is definitely longer than 10 characters",
		}
		itemJSON, _ := json.Marshal(item)

		// Should still render, just log a warning
		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if len(result) <= 10 {
			t.Error("Expected result to exceed character limit (should still render)")
		}
	})
}

func TestGetDefaultTemplate(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		tmpl := GetDefaultTemplate()
		if tmpl == "" {
			t.Error("Expected non-empty default template")
		}
	})

	t.Run("returned template is valid", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte(GetDefaultTemplate()), 0644)
		if err != nil {
			t.Fatalf("Failed to write default template: %v", err)
		}

		// Should be able to create renderer with default template
		_, err = New(tmplPath, 500)
		if err != nil {
			t.Errorf("Default template is invalid: %v", err)
		}
	})
}

func TestIntegration(t *testing.T) {
	t.Run("complete flow with real gofeed Item", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		tmplContent := `{{.Item.Title}}
{{.Item.Link}}
{{if .Item.Description}}{{.Item.Description}}{{end}}`
		err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{
			Title:       "Example Blog Post",
			Link:        "https://example.com/post/1",
			Description: "This is a description of the blog post.",
		}
		itemJSON, _ := json.Marshal(item)

		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		// Verify output contains expected parts
		if !contains(result, "Example Blog Post") {
			t.Error("Result should contain title")
		}
		if !contains(result, "https://example.com/post/1") {
			t.Error("Result should contain link")
		}
		if !contains(result, "This is a description") {
			t.Error("Result should contain description")
		}
	})

	t.Run("complex template with conditionals", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		tmplContent := `{{.Item.Title}}
{{if .Item.Author}}By: {{.Item.Author.Name}}{{end}}
{{.Item.Link}}`
		err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}

		renderer, err := New(tmplPath, 500)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		item := &gofeed.Item{
			Title:  "Test Post",
			Link:   "https://example.com/test",
			Author: &gofeed.Person{Name: "John Doe"},
		}
		itemJSON, _ := json.Marshal(item)

		result, err := renderer.Render(itemJSON)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if !contains(result, "By: John Doe") {
			t.Error("Result should contain author name")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
