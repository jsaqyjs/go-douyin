package dto

import "time"

// 发表评论请求
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,max=1000"`
	ParentID uint64 `json:"parent_id"`
}

// 评论对外响应
type CommentResponse struct {
	ID        uint64    `json:"id"`
	VideoID   uint64    `json:"video_id"`
	UserID    uint64    `json:"user_id"`
	ParentID  uint64    `json:"parent_id"`
	Content   string    `json:"content"`
	LikeCount int64     `json:"like_count"`
	CreatedAt time.Time `json:"created_at"`
}

// 评论分页响应
type CommentListResponse struct {
	List     []CommentResponse `json:"list"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int64             `json:"total"`
}
