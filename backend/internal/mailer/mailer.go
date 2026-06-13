// Package mailer sends transactional emails, such as magic-link sign-in
// links.
package mailer

import "context"

// Mailer sends transactional emails.
type Mailer interface {
	// SendMagicLink emails a sign-in link to toEmail.
	SendMagicLink(ctx context.Context, toEmail, link string) error
}

const (
	magicLinkSubject = "Your Massa sign-in link"
)

func magicLinkBody(link string) (text, html string) {
	text = "Click the link below to sign in to Massa. This link expires in 15 minutes and can only be used once.\n\n" + link
	html = `<p>Click the link below to sign in to Massa. This link expires in 15 minutes and can only be used once.</p><p><a href="` + link + `">` + link + `</a></p>`
	return text, html
}
