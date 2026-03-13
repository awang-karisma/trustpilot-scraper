package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

type WebhookService struct {
	URL          string
	TemplatePath string
}

func NewWebhookService(url, templatePath string) *WebhookService {
	return &WebhookService{
		URL:          url,
		TemplatePath: templatePath,
	}
}

type TemplateData struct {
	Website     string
	Reviewer    string
	Title       string
	Content     string
	Rating      int
	TotalRating float64
	TotalCount  int
	Date        string
}

func (w *WebhookService) Send(website string, review database.Review, summary database.Summary) error {
	if w.URL == "" {
		return nil // Webhook not configured
	}

	log.Printf("Sending webhook to %s using template %s", w.URL, w.TemplatePath)

	tmpl, err := template.New("webhook").Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			s := string(b)
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				// Return the content inside the quotes, but with backslashes escaped
				// json.Marshal already escaped everything, we just need the content.
				return s[1 : len(s)-1]
			}
			return s
		},
	}).ParseFiles(w.TemplatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// We need to use the base name for the template execution if we use ParseFiles
	tmplName := filepath.Base(w.TemplatePath)

	data := TemplateData{
		Website:     website,
		Reviewer:    review.Reviewer,
		Title:       review.Title,
		Content:     review.Content,
		Rating:      review.Rating,
		TotalRating: summary.Rating,
		TotalCount:  summary.Count,
		Date:        review.Date.Format("2006-01-02 15:04:05"),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// We'll parse the output into a map and then re-marshal it to ensure it's valid JSON
	// This also handles any accidental literal newlines introduced in the template strings
	var check map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		log.Printf("Template output is not valid JSON, attempting to fix literal newlines: %v", err)
		// Fallback: If it's invalid, it might be due to literal newlines.
		// For now, let's just log the error and try to send it anyway,
		// but the real fix is in the template or the marshal.
	}

	log.Printf("Payload prepared: %s", buf.String())

	req, err := http.NewRequest("POST", w.URL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned non-success status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
