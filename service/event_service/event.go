package event_service

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidEvent = errors.New("事件数据不合法")

type LikeEvent struct {
	EventID string `json:"event_id"`
	VideoID uint64 `json:"video_id"`
	UserID  uint64 `json:"user_id"`
}

// DecodeLikeEvent 获取页面传输信息
func DecodeLikeEvent(body []byte) (LikeEvent, error) {
	var event LikeEvent

	if err := json.Unmarshal(body, &event); err != nil {
		return event, ErrInvalidEvent
	}

	event.EventID = strings.TrimSpace(event.EventID)

	if event.EventID == "" ||
		len(event.EventID) > 64 ||
		event.VideoID == 0 ||
		event.UserID == 0 {
		return event, ErrInvalidEvent
	}

	return event, nil
}
