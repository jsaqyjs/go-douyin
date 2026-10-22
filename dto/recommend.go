package dto

type RecommendVideoItem struct {
	ID           uint64  `json:"id"`
	AuthorID     uint64  `json:"author_id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Category     string  `json:"category"`
	PlayURL      string  `json:"play_url"`
	CoverURL     string  `json:"cover_url"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	Score        float64 `json:"score"`
}

type RecommendFeedResponse struct {
	List       []RecommendVideoItem `json:"list"`
	NextCursor string               `json:"next_cursor"`
	HasMore    bool                 `json:"has_more"`
	ColdStart  bool                 `json:"cold_start"`
}
