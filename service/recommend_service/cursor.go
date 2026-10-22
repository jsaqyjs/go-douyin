package recommend_service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"time"
)

var ErrInvalidCursor = errors.New("推荐游标不合法")

type recommendCursor struct {
	UserID uint64    `json:"user_id"`
	AsOf   time.Time `json:"as_of"`
	Score  float64   `json:"score"`
	ID     uint64    `json:"id"`
}

func encodeCursor(c recommendCursor) (string, error) {
	if c.UserID == 0 ||
		c.ID == 0 ||
		c.AsOf.IsZero() ||
		math.IsNaN(c.Score) ||
		math.IsInf(c.Score, 0) {
		return "", ErrInvalidCursor
	}

	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeCursor(
	raw string,
	userID uint64,
) (recommendCursor, error) {
	var c recommendCursor

	if raw == "" || len(raw) > 1024 {
		return c, ErrInvalidCursor
	}

	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return c, ErrInvalidCursor
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, ErrInvalidCursor
	}

	if c.UserID != userID ||
		c.ID == 0 ||
		c.AsOf.IsZero() ||
		math.IsNaN(c.Score) ||
		math.IsInf(c.Score, 0) {
		return c, ErrInvalidCursor
	}

	// 游标最多使用30分钟。
	now := time.Now()
	if c.AsOf.After(now.Add(time.Minute)) ||
		now.Sub(c.AsOf) > 30*time.Minute {
		return c, ErrInvalidCursor
	}

	return c, nil
}
