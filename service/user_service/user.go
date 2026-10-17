package user_service

import (
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUsernameExists = errors.New("用户名已存在")
	ErrInvalidLogin   = errors.New("用户名或密码错误")
	ErrUserNotFound   = errors.New("用户不存在")
)

// 将数据库用户转换成对外响应
func ToUserResponse(user models.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	}
}

// 注册用户
func Register(req dto.RegisterRequest) (dto.UserResponse, error) {
	var empty dto.UserResponse

	// bcrypt 最多处理72字节密码
	if len([]byte(req.Password)) > 72 {
		return empty, errors.New("密码不能超过72字节")
	}

	// 生成密码哈希
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return empty, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 构造数据库模型
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		Nickname:     req.Nickname,
	}

	// 写入 MySQL
	if err := database.DB.Create(&user).Error; err != nil {
		var mysqlErr *mysql.MySQLError

		// MySQL 1062 表示唯一索引冲突
		if errors.As(err, &mysqlErr) &&
			mysqlErr.Number == 1062 {
			return empty, ErrUsernameExists
		}

		// 兼容开启 TranslateError 的 GORM 配置
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return empty, ErrUsernameExists
		}

		return empty, fmt.Errorf("创建用户失败: %w", err)
	}

	return ToUserResponse(user), nil
}

// 用户登录
func Login(req dto.LoginRequest) (dto.LoginResponse, error) {
	var result dto.LoginResponse
	var user models.User

	// 根据用户名查找用户
	err := database.DB.
		Where("username = ?", req.Username).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, ErrInvalidLogin
	}

	if err != nil {
		return result, fmt.Errorf("查询用户失败: %w", err)
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return result, ErrInvalidLogin
	}

	// 生成JWT
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return result, fmt.Errorf("生成Token失败: %w", err)
	}

	result.Token = token
	result.User = ToUserResponse(user)

	return result, nil
}

// GetUserByID 查询当前用户
func GetUserByID(userID uint64) (
	dto.UserResponse,
	error,
) {
	var user models.User

	err := database.DB.First(&user, userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.UserResponse{}, ErrUserNotFound
	}

	if err != nil {
		return dto.UserResponse{}, fmt.Errorf(
			"查询用户失败: %w",
			err,
		)
	}

	return ToUserResponse(user), nil
}
