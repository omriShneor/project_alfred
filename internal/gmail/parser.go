package gmail

import (
	"regexp"
	"strings"
)

// HTMLToText converts HTML content to plain text
// This is a simple implementation for email content extraction
func HTMLToText(html string) string {
	if html == "" {
		return ""
	}

	text := html

	// Replace common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")

	// Remove script and style tags with content
	scriptRegex := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	text = scriptRegex.ReplaceAllString(text, "")

	styleRegex := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	text = styleRegex.ReplaceAllString(text, "")

	// Replace br tags with newlines
	brRegex := regexp.MustCompile(`(?i)<br\s*/?>`)
	text = brRegex.ReplaceAllString(text, "\n")

	// Replace p, div, tr tags with double newlines for paragraph breaks
	blockRegex := regexp.MustCompile(`(?i)</?(p|div|tr|table|article|section|header|footer)[^>]*>`)
	text = blockRegex.ReplaceAllString(text, "\n\n")

	// Replace li tags with bullet points
	liRegex := regexp.MustCompile(`(?i)<li[^>]*>`)
	text = liRegex.ReplaceAllString(text, "\n- ")

	// Remove all remaining HTML tags
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Collapse multiple newlines to maximum of two
	multiNewlineRegex := regexp.MustCompile(`\n{3,}`)
	text = multiNewlineRegex.ReplaceAllString(text, "\n\n")

	// Collapse multiple spaces
	multiSpaceRegex := regexp.MustCompile(`[ \t]+`)
	text = multiSpaceRegex.ReplaceAllString(text, " ")

	// Trim whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// ExtractSenderEmail extracts just the email address from a "From" header
// e.g., "John Doe <john@example.com>" -> "john@example.com"
func ExtractSenderEmail(from string) string {
	// Try to extract email from angle brackets
	emailRegex := regexp.MustCompile(`<([^>]+)>`)
	matches := emailRegex.FindStringSubmatch(from)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no angle brackets, assume the whole thing is an email
	return strings.TrimSpace(from)
}

// ExtractSenderName extracts the display name from a "From" header
// e.g., "John Doe <john@example.com>" -> "John Doe"
func ExtractSenderName(from string) string {
	// Check for angle brackets format
	if idx := strings.Index(from, "<"); idx > 0 {
		name := strings.TrimSpace(from[:idx])
		// Remove quotes if present
		name = strings.Trim(name, "\"'")
		return name
	}

	// If no name found, use the email address
	return ExtractSenderEmail(from)
}

// ExtractDomain extracts the domain from an email address
// e.g., "john@example.com" -> "example.com"
func ExtractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// TruncateText truncates text to a maximum length, adding ellipsis if needed
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// CleanEmailBody cleans and normalizes email body text for processing
func CleanEmailBody(body string) string {
	// First convert HTML if present
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		body = HTMLToText(body)
	}

	// Remove common email signature patterns
	sigPatterns := []string{
		"-- \n",          // Standard signature delimiter
		"---\n",          // Alternative delimiter
		"Sent from my",   // Mobile signatures
		"Get Outlook for", // Outlook mobile signature
	}

	for _, pattern := range sigPatterns {
		if idx := strings.Index(body, pattern); idx > 0 {
			body = body[:idx]
		}
	}

	// Remove excessive quoted text (previous email in thread)
	quotedRegex := regexp.MustCompile(`(?m)^>.*$`)
	body = quotedRegex.ReplaceAllString(body, "")

	// Clean up any resulting excessive whitespace
	multiNewlineRegex := regexp.MustCompile(`\n{3,}`)
	body = multiNewlineRegex.ReplaceAllString(body, "\n\n")

	return strings.TrimSpace(body)
}
