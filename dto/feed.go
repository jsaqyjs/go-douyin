package dto

// 关注Feed响应
type FollowingFeedResponse struct {
	List       []VideoResponse `json:"list"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}
