package dto

type HotVideoItem struct {
	ID           uint64  `json:"id"`
	AuthorID     uint64  `json:"author_id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	PlayURL      string  `json:"play_url"`
	CoverURL     string  `json:"cover_url"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	HotScore     float64 `json:"hot_score"`
}

type HotVideoResponse struct {
	List   []HotVideoItem `json:"list"`
	Source string         `json:"source"`
}
