package comment_service

import (
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

var (
	ErrInvalidPage     = errors.New("分页参数不合法")
	ErrVideoNotFound   = errors.New("视频不存在或不可评论")
	ErrInvalidParent   = errors.New("父评论无效")
	ErrInvalidContent  = errors.New("评论内容不能为空或过长")
	ErrCommentNotFound = errors.New("评论不存在")
	ErrForbidden       = errors.New("无权删除该评论")
)

func toCommentResponse(c models.Comment) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        c.ID,
		VideoID:   c.VideoID,
		UserID:    c.UserID,
		ParentID:  c.ParentID,
		Content:   c.Content,
		LikeCount: c.LikeCount,
		CreatedAt: c.CreatedAt,
	}
}

// CreateComment 发表或回复评论
func CreateComment(
	userID, videoID uint64,
	req dto.CreateCommentRequest,
) (dto.CommentResponse, error) {
	// 内容是否正确 防止有 “  ”
	content := strings.TrimSpace(req.Content)
	if content == "" || len([]rune(content)) > 1000 {
		return dto.CommentResponse{}, ErrInvalidContent
	}
	var comment models.Comment
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 锁定视频，协调评论数量更新及删除操作
		var video models.Video

		err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
		}).
			Where(
				"id = ? AND status = ?",
				videoID,
				models.VideoStatusPublished,
			).
			First(&video).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		if err != nil {
			return err
		}
		// 2. 回复时校验父评论 上级评论是否存在
		if req.ParentID != 0 {
			var parent models.Comment

			err := tx.Where(
				"id = ? AND video_id = ? AND parent_id = 0",
				req.ParentID,
				videoID,
			).First(&parent).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidParent
			}
			if err != nil {
				return err
			}

		}
		// 3. 创建评论
		comment = models.Comment{
			UserID:   userID,
			VideoID:  videoID,
			ParentID: req.ParentID,
			Content:  content,
		}
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		// 4. 视频评论数 +1
		result := tx.Model(&models.Video{}).
			Where("id = ?", videoID).
			UpdateColumn(
				"comment_count",
				gorm.Expr("comment_count + 1"),
			)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("评论计数更新失败")
		}

		return nil
	})
	if err != nil {
		return dto.CommentResponse{},
			fmt.Errorf("发表评论失败: %w", err)
	}

	return toCommentResponse(comment), nil
}

// ListComments 获取某个视频的一级评论
func ListComments(
	videoID uint64,
	page, pageSize int,
) (dto.CommentListResponse, error) {
	return listByCondition(
		videoID, 0, page, pageSize, true,
	)
}

// ListReplies 获取某个视频中一级评论的回复
func ListReplies(
	parentID uint64,
	page, pageSize int,
) (dto.CommentListResponse, error) {
	var parent models.Comment

	err := database.DB.Where(
		"id = ? AND parent_id = 0",
		parentID,
	).First(&parent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.CommentListResponse{}, ErrCommentNotFound
	}
	if err != nil {
		return dto.CommentListResponse{}, err
	}
	return listByCondition(
		parent.VideoID, parentID, page, pageSize, false,
	)
}

// listByCondition 共用分页查询逻辑
func listByCondition(
	videoID, parentID uint64,
	page, pageSize int,
	checkVideo bool,
) (dto.CommentListResponse, error) {

	result := dto.CommentListResponse{
		List:     make([]dto.CommentResponse, 0),
		Page:     page,
		PageSize: pageSize,
	}
	//	分页规则
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return result, ErrInvalidPage
	}

	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		return result, ErrInvalidPage
	}

	// 公开列表必须检查视频是否正常
	// 回复列表也调用同样的检查，防止下架视频泄露评论
	_ = checkVideo
	var video models.Video
	// 视频是否正常
	err := database.DB.Where(
		"id = ? AND status = ?",
		videoID,
		models.VideoStatusPublished,
	).First(&video).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, ErrVideoNotFound
	}
	if err != nil {
		return result, err
	}
	// 视频上传后进行 一级评论的统计
	query := database.DB.Model(&models.Comment{}).
		Where("video_id = ? AND parent_id = ?", videoID, parentID)

	if err := query.Count(&result.Total).Error; err != nil {
		return result, err
	}
	// 对一级评论做分页
	var comments []models.Comment

	err = query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&comments).Error

	if err != nil {
		return result, err
	}
	// 入值完成后进行评论收集
	for _, comment := range comments {
		result.List = append(
			result.List,
			toCommentResponse(comment),
		)
	}

	return result, nil
}

// DeleteComment 只能删除自己的评论
func DeleteComment(userID, commentID uint64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 先读取评论，确定所属视频
		var target models.Comment

		err := tx.First(&target, commentID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		if err != nil {
			return err
		}

		// 2. 锁定视频行，与发表评论事务协调
		var video models.Video

		if err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
		}).First(&video, target.VideoID).Error; err != nil {
			return err
		}

		// 3. 锁住视频后重新检查评论，避免并发删除
		err = tx.First(&target, commentID).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		if err != nil {
			return err
		}

		// 4. 权限校验
		if target.UserID != userID {
			return ErrForbidden
		}

		// 5. 一级评论连同其回复一起删除
		var removed int64

		if target.ParentID == 0 {
			result := tx.Where(
				"parent_id = ? AND video_id = ?",
				target.ID, target.VideoID,
			).Delete(&models.Comment{})

			if result.Error != nil {
				return result.Error
			}
			removed += result.RowsAffected
		}

		result := tx.Where(
			"id = ?", target.ID,
		).Delete(&models.Comment{})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrCommentNotFound
		}

		removed++

		// 6. 视频评论数减去实际删除数量
		result = tx.Model(&models.Video{}).
			Where(
				"id = ? AND comment_count >= ?",
				target.VideoID, removed,
			).
			UpdateColumn(
				"comment_count",
				gorm.Expr("comment_count - ?", removed),
			)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("评论计数异常，事务回滚")
		}

		return nil
	})
}
