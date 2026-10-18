package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// 游标内部的数据结构
type FeedCursor struct {
	Time time.Time `json:"time"`
	ID   uint64    `json:"id"`
}

// EncodeFeedCursor 编码游标
func EncodeFeedCursor(
	createdAt time.Time,
	videoID uint64,
) (string, error) {
	data := FeedCursor{
		Time: createdAt,
		ID:   videoID,
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeFeedCursor 解析游标
func DecodeFeedCursor(cursor string) (FeedCursor, error) {
	var result FeedCursor

	if cursor == "" || len(cursor) > 512 {
		return result, errors.New("游标不合法")
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return result, errors.New("游标解码失败")
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		return result, errors.New("游标内容不合法")
	}

	if result.ID == 0 || result.Time.IsZero() {
		return result, errors.New("游标字段无效")
	}

	return result, nil
}
