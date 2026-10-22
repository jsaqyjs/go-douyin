package dto

type FavoriteVideoItem struct {
	ID           uint64 `json:"id"`
	AuthorID     uint64 `json:"author_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	PlayURL      string `json:"play_url"`
	CoverURL     string `json:"cover_url"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
	FavoritedAt  string `json:"favorited_at"`
}

type FavoriteListResponse struct {
	List    []FavoriteVideoItem `json:"list"`
	Page    int                 `json:"page"`
	Limit   int                 `json:"limit"`
	Total   int64               `json:"total"`
	HasMore bool                `json:"has_more"`
}
