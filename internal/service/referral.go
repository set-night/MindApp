package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/set-night/mindapp/internal/repository/sqlc"
)

const (
	referralCodeLength  = 6
	referralCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

func generateReferralCode() (string, error) {
	code := make([]byte, referralCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(referralCodeCharset))))
		if err != nil {
			return "", fmt.Errorf("random int: %w", err)
		}
		code[i] = referralCodeCharset[n.Int64()]
	}
	return string(code), nil
}

func generateUniqueReferralCode(ctx context.Context, queries *sqlc.Queries) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateReferralCode()
		if err != nil {
			return "", err
		}
		_, err = queries.GetUserByReferralCode(ctx, code)
		if err == pgx.ErrNoRows {
			return code, nil
		}
		if err != nil {
			return "", fmt.Errorf("check referral code: %w", err)
		}
	}
	return "", fmt.Errorf("failed to generate unique referral code after 10 attempts")
}
