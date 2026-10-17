package dto

import "time"

// PublishVideoRequest 发布视频请求
type PublishVideoRequest struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description"`
	PlayURL     string `json:"play_url" binding:"required,max=1024,url"`
	CoverURL    string `json:"cover_url" binding:"omitempty,max=1024,url"`
}

// VideoResponse 对外返回的视频信息
type VideoResponse struct {
	ID           uint64    `json:"id"`
	AuthorID     uint64    `json:"author_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	PlayURL      string    `json:"play_url"`
	CoverURL     string    `json:"cover_url"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// VideoListResponse 分页响应
type VideoListResponse struct {
	List     []VideoResponse `json:"list"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
}
