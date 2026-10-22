package dto

type SearchVideoItem struct {
	ID           uint64 `json:"id"`
	AuthorID     uint64 `json:"author_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	PlayURL      string `json:"play_url"`
	CoverURL     string `json:"cover_url"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
}

type VideoSearchResponse struct {
	List       []SearchVideoItem `json:"list"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}
