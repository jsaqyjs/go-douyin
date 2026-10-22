package hot_service

import (
	"math"
	"time"
)

// CalculateHotScore 计算视频热度
func CalculateHotScore(
	likes int64,
	comments int64,
	createdAt time.Time,
	now time.Time,
) float64 {
	if likes < 0 {
		likes = 0
	}
	if comments < 0 {
		comments = 0
	}

	hours := now.Sub(createdAt).Hours()
	if hours < 0 {
		hours = 0
	}

	interactions := float64(likes) +
		3*float64(comments)

	return interactions /
		math.Pow(1+hours/24, 1.5)
}
