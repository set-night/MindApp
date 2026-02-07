package telegram

import (
	"strings"
	"unicode/utf8"
)

// SplitMessage splits a message into chunks of maxLen characters,
// trying to split at newlines when possible.
func SplitMessage(text string, maxLen int) []string {
	if utf8.RuneCountInString(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if utf8.RuneCountInString(text) <= maxLen {
			parts = append(parts, text)
			break
		}

		// Find split point
		runes := []rune(text)
		splitAt := maxLen

		// Try to split at a newline
		chunk := string(runes[:maxLen])
		lastNewline := strings.LastIndex(chunk, "\n")
		if lastNewline > maxLen/2 {
			splitAt = lastNewline + 1
		}

		parts = append(parts, string(runes[:splitAt]))
		text = string(runes[splitAt:])
	}

	return parts
}

// IsValidMarkdownV2 checks if the text has balanced markdown formatting.
func IsValidMarkdownV2(text string) bool {
	// Check for balanced code blocks
	codeBlockCount := strings.Count(text, "```")
	if codeBlockCount%2 != 0 {
		return false
	}

	// Check for balanced inline code
	inlineCodeCount := 0
	inCodeBlock := false
	for i := 0; i < len(text); i++ {
		if i+2 < len(text) && text[i:i+3] == "```" {
			inCodeBlock = !inCodeBlock
			i += 2
			continue
		}
		if !inCodeBlock && text[i] == '`' {
			inlineCodeCount++
		}
	}
	if inlineCodeCount%2 != 0 {
		return false
	}

	return true
}

// FixMarkdown attempts to fix common markdown issues.
func FixMarkdown(text string) string {
	// Fix unclosed code blocks
	codeBlockCount := strings.Count(text, "```")
	if codeBlockCount%2 != 0 {
		text += "\n```"
	}

	// Fix unclosed inline code (outside of code blocks)
	result := fixInlineCode(text)
	return result
}

func fixInlineCode(text string) string {
	var builder strings.Builder
	inCodeBlock := false
	inlineOpen := false

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		// Check for code blocks
		if i+2 < len(runes) && string(runes[i:i+3]) == "```" {
			if inlineOpen {
				builder.WriteRune('`')
				inlineOpen = false
			}
			inCodeBlock = !inCodeBlock
			builder.WriteString("```")
			i += 2
			continue
		}

		if !inCodeBlock && runes[i] == '`' {
			inlineOpen = !inlineOpen
		}

		builder.WriteRune(runes[i])
	}

	if inlineOpen {
		builder.WriteRune('`')
	}

	return builder.String()
}
