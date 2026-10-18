package follow_service

import (
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrFollowSelf   = errors.New("不能关注自己")
	ErrInvalidPage  = errors.New("分页参数不合法")
)

// checkUser 检查用户是否存在
func checkUser(userID uint64) error {
	var user models.User

	err := database.DB.
		Select("id").
		First(&user, userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}

	return err
}

// FollowUser 关注用户
// changed=true 表示本次确实创建了关注关系
func FollowUser(
	followerID, followingID uint64,
) (bool, error) {

	// 不允许关注自己
	if followerID == followingID {
		return false, ErrFollowSelf
	}

	// 确认目标用户存在
	if err := checkUser(followingID); err != nil {
		return false, err
	}

	follow := models.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}

	// 如果联合唯一索引冲突，则不插入
	result := database.DB.Clauses(
		clause.OnConflict{DoNothing: true},
	).Create(&follow)

	if result.Error != nil {
		return false, fmt.Errorf("关注失败: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

// UnfollowUser 取消关注
func UnfollowUser(
	followerID, followingID uint64,
) (bool, error) {
	if followerID == followingID {
		return false, ErrFollowSelf
	}
	result := database.DB.Where(
		"follower_id = ? AND following_id = ?",
		followerID,
		followingID,
	).Delete(&models.Follow{})

	if result.Error != nil {
		return false, fmt.Errorf("取消关注失败: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// IsFollowing 查询是否关注目标用户
func IsFollowing(
	followerID, followingID uint64,
) (bool, error) {
	if err := checkUser(followingID); err != nil {
		return false, err
	}

	var count int64

	err := database.DB.Model(&models.Follow{}).
		Where(
			"follower_id = ? AND following_id = ?",
			followerID,
			followingID,
		).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ListFollowing 查询某人关注的用户
func ListFollowing(
	userID uint64,
	page, pageSize int,
) (dto.FollowListResponse, error) {

	return listUsers(userID, page, pageSize, true)
}

// ListFollowers 查询某人的粉丝
func ListFollowers(
	userID uint64,
	page, pageSize int,
) (dto.FollowListResponse, error) {

	return listUsers(userID, page, pageSize, false)
}

// listUsers 公共分页查询
func listUsers(
	userID uint64,
	page, pageSize int,
	following bool,
) (dto.FollowListResponse, error) {
	result := dto.FollowListResponse{
		List:     make([]dto.FollowUserResponse, 0),
		Page:     page,
		PageSize: pageSize,
	}
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return result, ErrInvalidPage
	}

	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		return result, ErrInvalidPage
	}

	if err := checkUser(userID); err != nil {
		return result, err
	}
	// 根据查询方向选择不同的关联
	condition := "f.follower_id = ?"
	joinCondition := "u.id = f.following_id"

	if !following {
		condition = "f.following_id = ?"
		joinCondition = "u.id = f.follower_id"
	}
	query := database.DB.Table("follow AS f").
		Joins("JOIN user AS u ON "+joinCondition).
		Where(condition, userID)
	if err := query.Count(&result.Total).Error; err != nil {
		return result, err
	}
	err := query.Select(
		"u.id, u.username, u.nickname, " +
			"u.avatar_url, f.created_at AS followed_at",
	).
		Order("f.created_at DESC").
		Order("f.id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&result.List).Error

	if err != nil {
		return result, fmt.Errorf("查询关注列表失败: %w", err)
	}

	return result, nil
}
