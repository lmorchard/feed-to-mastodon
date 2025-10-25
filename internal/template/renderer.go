package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

// Renderer handles template rendering for Mastodon posts.
type Renderer struct {
	tmpl           *template.Template
	characterLimit int
	feed           *gofeed.Feed
}

// TemplateData holds the data passed to templates.
type TemplateData struct {
	Item *gofeed.Item
	Feed *gofeed.Feed
}

// New creates a new Renderer with the specified template file and character limit.
func New(templatePath string, characterLimit int) (*Renderer, error) {
	// Read template file
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template with custom functions
	tmpl, err := template.New("post").Funcs(template.FuncMap{
		"truncate":       truncate,
		"htmltomarkdown": htmlToMarkdown,
	}).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Renderer{
		tmpl:           tmpl,
		characterLimit: characterLimit,
	}, nil
}

// truncate truncates a string to the specified maximum length.
// If the string is longer than maxLen, it's truncated and "..." is appended.
// Handles UTF-8 properly by counting runes, not bytes.
// maxLen can be passed as int or any numeric type.
func truncate(s string, maxLen interface{}) string {
	// Convert maxLen to int
	var limit int
	switch v := maxLen.(type) {
	case int:
		limit = v
	case int64:
		limit = int(v)
	case float64:
		limit = int(v)
	default:
		return s
	}

	if limit <= 0 {
		return ""
	}

	// Count runes to handle UTF-8 properly
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	// Truncate and add ellipsis
	if limit <= 3 {
		return string(runes[:limit])
	}

	return string(runes[:limit-3]) + "..."
}

// htmlToMarkdown converts HTML content to markdown format.
// If conversion fails, the original HTML is returned.
func htmlToMarkdown(html string) string {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		logrus.Warnf("Failed to convert HTML to markdown: %v", err)
		return html
	}
	return markdown
}

// SetFeed sets the feed metadata for use in templates.
func (r *Renderer) SetFeed(feed *gofeed.Feed) {
	r.feed = feed
}

// Render renders the template with the given entry data.
func (r *Renderer) Render(entryJSON []byte) (string, error) {
	// Unmarshal entry JSON into gofeed.Item
	var item gofeed.Item
	if err := json.Unmarshal(entryJSON, &item); err != nil {
		return "", fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	// Create template data
	data := TemplateData{
		Item: &item,
		Feed: r.feed,
	}

	// Execute template
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	rendered := buf.String()

	// Check character limit and warn if exceeded
	runeCount := utf8.RuneCountInString(rendered)
	if runeCount > r.characterLimit {
		logrus.Warnf("Rendered post exceeds character limit: %d > %d", runeCount, r.characterLimit)
	}

	return rendered, nil
}

// GetDefaultTemplate returns a simple default template.
func GetDefaultTemplate() string {
	return `{{.Item.Title}}

{{if .Item.Description}}{{truncate .Item.Description 100}}{{else if .Item.Content}}{{truncate .Item.Content 100}}{{end}}

{{.Item.Link}}

{{range .Item.Categories}}#{{.}} {{end}}

Via {{.Feed.Title}}`
}
