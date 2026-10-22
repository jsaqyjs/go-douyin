package video_search_service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidKeyword = errors.New("搜索关键词必须是1到50个字符")
	ErrInvalidCursor  = errors.New("搜索游标不合法")
)

type searchCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uint64    `json:"id"`
	Keyword   string    `json:"keyword"`
}

// NormalizeKeyword 去掉首尾空白并校验长度
func NormalizeKeyword(raw string) (string, error) {
	keyword := strings.TrimSpace(raw)

	if !utf8.ValidString(keyword) {
		return "", ErrInvalidKeyword
	}

	n := utf8.RuneCountInString(keyword)
	if n < 1 || n > 50 {
		return "", ErrInvalidKeyword
	}

	return keyword, nil
}

// EscapeLike 将用户输入作为字面文本匹配。
// 用 ! 作为SQL LIKE的转义字符。
func EscapeLike(keyword string) string {
	replacer := strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	)
	return replacer.Replace(keyword)
}
func encodeCursor(
	createdAt time.Time,
	id uint64,
	keyword string,
) (string, error) {
	cursor := searchCursor{
		CreatedAt: createdAt,
		ID:        id,
		Keyword:   keyword,
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}
func decodeCursor(
	raw string,
	keyword string,
) (searchCursor, error) {
	var cursor searchCursor

	if raw == "" || len(raw) > 2048 {
		return cursor, ErrInvalidCursor
	}

	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor, ErrInvalidCursor
	}

	if err := json.Unmarshal(data, &cursor); err != nil {
		return cursor, ErrInvalidCursor
	}

	if cursor.CreatedAt.IsZero() ||
		cursor.ID == 0 ||
		cursor.Keyword != keyword {
		return cursor, ErrInvalidCursor
	}

	return cursor, nil
}

// SearchVideos 根据关键词查询公开视频。
// 排序使用(created_at DESC, id DESC)。
func SearchVideos(
	ctx context.Context,
	rawKeyword string,
	limit int,
	cursorString string,
) (dto.VideoSearchResponse, error) {
	result := dto.VideoSearchResponse{
		List: make([]dto.SearchVideoItem, 0),
	}

	keyword, err := NormalizeKeyword(rawKeyword)
	if err != nil {
		return result, err
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	var cursor searchCursor

	if cursorString != "" {
		cursor, err = decodeCursor(
			cursorString, keyword,
		)
		if err != nil {
			return result, err
		}
	}

	// 将关键词转义，避免%和_被当成通配符
	pattern := "%" + EscapeLike(keyword) + "%"
	query := database.DB.WithContext(ctx).
		Model(&models.Video{}).
		Where("status = ?", 1).
		Where(
			`(title LIKE ? ESCAPE '!'
              OR description LIKE ? ESCAPE '!')`,
			pattern, pattern,
		)
	if cursorString != "" {
		query = query.Where(
			`(created_at < ? OR
              (created_at = ? AND id < ?))`,
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	var videos []models.Video

	err = query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit + 1).
		Find(&videos).Error

	if err != nil {
		return result, err
	}
	result.HasMore = len(videos) > limit

	if result.HasMore {
		videos = videos[:limit]
	}

	for _, video := range videos {
		result.List = append(
			result.List,
			dto.SearchVideoItem{
				ID:           video.ID,
				AuthorID:     video.AuthorID,
				Title:        video.Title,
				Description:  video.Description,
				PlayURL:      video.PlayURL,
				CoverURL:     video.CoverURL,
				LikeCount:    video.LikeCount,
				CommentCount: video.CommentCount,
				CreatedAt: video.CreatedAt.Format(
					time.RFC3339Nano,
				),
			},
		)
	}

	if result.HasMore && len(videos) > 0 {
		last := videos[len(videos)-1]

		result.NextCursor, err = encodeCursor(
			last.CreatedAt,
			last.ID,
			keyword,
		)
		if err != nil {
			return dto.VideoSearchResponse{}, err
		}
	}

	return result, nil
}
