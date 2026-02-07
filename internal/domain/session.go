package domain

import (
	"time"
)

type ChatSession struct {
	ID          int64
	UserID      int64
	Model       string
	Temperature float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SessionMessage struct {
	ID        int64
	SessionID int64
	Role      string
	Text      string
	Images    []string
	IsSystem  bool
	CreatedAt time.Time
	Files     []MessageFile
}

type MessageFile struct {
	ID        int64
	MessageID int64
	FileType  string // image, video, audio, document
	URL       string
	Name      string
}
