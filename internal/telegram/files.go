package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-telegram/bot"
)

// DownloadFile downloads a file from Telegram by file ID.
func DownloadFile(ctx context.Context, b *bot.Bot, fileID string) ([]byte, string, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, "", fmt.Errorf("get file: %w", err)
	}

	fileURL := b.FileDownloadLink(file)

	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read file data: %w", err)
	}

	return data, file.FilePath, nil
}

// GetFileURL returns the download URL for a Telegram file.
func GetFileURL(ctx context.Context, b *bot.Bot, fileID string) (string, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("get file: %w", err)
	}
	return b.FileDownloadLink(file), nil
}
