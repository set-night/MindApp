package service

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// pgTimestamptzToTime converts pgtype.Timestamptz to time.Time.
func pgTimestamptzToTime(ts pgtype.Timestamptz) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}

// pgTimestamptzToTimePtr converts pgtype.Timestamptz to *time.Time.
func pgTimestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if ts.Valid {
		t := ts.Time
		return &t
	}
	return nil
}

// timeToPgTimestamptz converts time.Time to pgtype.Timestamptz.
func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// timePtrToPgTimestamptz converts *time.Time to pgtype.Timestamptz.
func timePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// decimalToFloat converts decimal.Decimal to float64.
func decimalToFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// int32PtrToIntPtr converts *int32 to *int.
func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

// intPtrToInt32Ptr converts *int to *int32.
func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}
