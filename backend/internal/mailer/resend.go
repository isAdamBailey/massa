package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const defaultResendAPIURL = "https://api.resend.com/emails"

// ResendMailer sends email via the Resend HTTP API.
type ResendMailer struct {
	apiKey     string
	from       string
	baseURL    string
	httpClient *http.Client
}

// NewResendMailer returns a Mailer that sends via the Resend API.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{apiKey: apiKey, from: from, baseURL: defaultResendAPIURL, httpClient: http.DefaultClient}
}

// SetBaseURL overrides the Resend API base URL, for testing.
func (m *ResendMailer) SetBaseURL(url string) {
	m.baseURL = url
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
}

// SendMagicLink implements Mailer.
func (m *ResendMailer) SendMagicLink(ctx context.Context, toEmail, link string) error {
	text, html := magicLinkBody(link)

	payload, err := json.Marshal(resendEmailRequest{
		From:    m.from,
		To:      []string{toEmail},
		Subject: magicLinkSubject,
		Text:    text,
		HTML:    html,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend: unexpected status %d", resp.StatusCode)
	}

	return nil
}
