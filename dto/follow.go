package dto

import "time"

type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

type FollowUserResponse struct {
	ID         uint64    `json:"id"`
	Username   string    `json:"username"`
	Nickname   string    `json:"nickname"`
	AvatarURL  string    `json:"avatar_url"`
	FollowedAt time.Time `json:"followed_at"`
}

type FollowListResponse struct {
	List     []FollowUserResponse `json:"list"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Total    int64                `json:"total"`
}
