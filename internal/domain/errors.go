package domain

import "errors"

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrPromoAlreadyUsed    = errors.New("promo already used by this user")
	ErrPromoMaxUses        = errors.New("promo max uses reached")
	ErrPromoNotFound       = errors.New("promo not found")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionLimitReached = errors.New("session limit reached")
	ErrModelNotFound       = errors.New("model not found")
	ErrActiveRequest       = errors.New("active request exists")
	ErrCooldown            = errors.New("request too soon")
	ErrMessageLimit        = errors.New("message limit exceeded")
	ErrBotBlocked          = errors.New("bot blocked by user")
	ErrUserNotFound        = errors.New("user not found")
	ErrGroupNotFound       = errors.New("group not found")
	ErrInvoiceNotFound     = errors.New("invoice not found")
	ErrTaskNotFound        = errors.New("task not found")
	ErrTaskAlreadyDone     = errors.New("task already completed")
	ErrNotGroupAdmin       = errors.New("not a group admin")
	ErrLowBalance          = errors.New("balance too low for this model")
	ErrInvalidAmount       = errors.New("invalid amount")
)
