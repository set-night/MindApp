package telegram

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// InlineButton creates a single inline keyboard button.
func InlineButton(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// URLButton creates a URL inline keyboard button.
func URLButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// InlineKeyboard creates an inline keyboard from rows of buttons.
func InlineKeyboard(rows ...[]models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// ButtonRow creates a row of inline buttons.
func ButtonRow(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return buttons
}

// PaginationRow creates a pagination row with prev/next buttons.
func PaginationRow(currentPage, totalPages int, callbackPrefix string) []models.InlineKeyboardButton {
	var row []models.InlineKeyboardButton

	if currentPage > 0 {
		row = append(row, InlineButton("⬅️", fmt.Sprintf("%s_%d", callbackPrefix, currentPage-1)))
	}

	row = append(row, InlineButton(
		fmt.Sprintf("%d/%d", currentPage+1, totalPages),
		"cur",
	))

	if currentPage < totalPages-1 {
		row = append(row, InlineButton("➡️", fmt.Sprintf("%s_%d", callbackPrefix, currentPage+1)))
	}

	return row
}
