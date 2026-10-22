package dto

// 用户主页响应
type UserProfileResponse struct {
	ID             uint64 `json:"id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url"`
	FollowingCount int64  `json:"following_count"`
	FollowerCount  int64  `json:"follower_count"`
	VideoCount     int64  `json:"video_count"`
	TotalLikes     int64  `json:"total_likes"`
}

// 作品列表中的视频信息
type ProfileVideoResponse struct {
	ID           uint64 `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	PlayURL      string `json:"play_url"`
	CoverURL     string `json:"cover_url"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
}

// 用户作品列表响应
type UserVideosResponse struct {
	List       []ProfileVideoResponse `json:"list"`
	NextCursor string                 `json:"next_cursor"`
	HasMore    bool                   `json:"has_more"`
}
