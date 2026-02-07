package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/service"
)

type ctxKey string

const (
	UserKey  ctxKey = "user"
	GroupKey ctxKey = "group"
)

// GetUser extracts user from context.
func GetUser(ctx context.Context) *domain.User {
	u, ok := ctx.Value(UserKey).(*domain.User)
	if !ok {
		return nil
	}
	return u
}

// GetGroup extracts group from context.
func GetGroup(ctx context.Context) *domain.Group {
	g, ok := ctx.Value(GroupKey).(*domain.Group)
	if !ok {
		return nil
	}
	return g
}

// UserLoader returns middleware that loads user/group into context.
func UserLoader(userService *service.UserService, groupService *service.GroupService, cfg interface{ IsAdmin(int64) bool }) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			var from *models.User
			var chatType string
			var chatID int64
			var chatUsername string
			var chatTitle string

			if update.Message != nil {
				from = update.Message.From
				chatType = string(update.Message.Chat.Type)
				chatID = update.Message.Chat.ID
				chatUsername = update.Message.Chat.Username
				chatTitle = update.Message.Chat.Title
			} else if update.CallbackQuery != nil {
				from = &update.CallbackQuery.From
				if update.CallbackQuery.Message.Message != nil {
					msg := update.CallbackQuery.Message.Message
					chatType = string(msg.Chat.Type)
					chatID = msg.Chat.ID
					chatUsername = msg.Chat.Username
					chatTitle = msg.Chat.Title
				}
			} else if update.PreCheckoutQuery != nil {
				from = update.PreCheckoutQuery.From
			}

			if from == nil {
				next(ctx, b, update)
				return
			}

			// Load user
			username := ""
			if from.Username != "" {
				username = from.Username
			}
			isAdmin := cfg.IsAdmin(from.ID)

			user, _, err := userService.FindOrCreate(ctx, from.ID, from.FirstName, username, "", isAdmin)
			if err == nil && user != nil {
				ctx = context.WithValue(ctx, UserKey, user)
			}

			// Load group if supergroup/group chat
			if chatType == "supergroup" || chatType == "group" {
				group, _, err := groupService.FindOrCreate(ctx, chatID, chatUsername, chatTitle)
				if err == nil && group != nil {
					ctx = context.WithValue(ctx, GroupKey, group)
				}
			}

			next(ctx, b, update)
		}
	}
}
