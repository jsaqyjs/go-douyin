package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/dto"
	"go-douyin/middleware"
	"go-douyin/service/user_service"
	"log/slog"
	"net/http"
)

// RegisterHandler 用户注册
func RegisterHandler(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "注册参数不合法",
			"data":    nil,
		})
		return
	}
	user, err := user_service.Register(req)
	if errors.Is(err, user_service.ErrUsernameExists) {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "用户名已存在",
			"data":    nil,
		})
		return
	}

	if err != nil {
		// 详细错误只记录在服务端，不直接返回给用户
		slog.Error("注册失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "注册失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "注册成功",
		"data":    user,
	})
}

// LoginHandler 用户登录
func LoginHandler(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "登录参数不合法",
			"data":    nil,
		})
		return
	}
	result, err := user_service.Login(req)
	if errors.Is(err, user_service.ErrInvalidLogin) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户名或密码错误",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("登录失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "登录失败",
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data":    result,
	})
}

// GetMeHandler 查询当前登录用户
func GetMeHandler(c *gin.Context) {

	// JWT中间件已把userID保存到Context
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    userID,
		})
		return
	}
	user, err := user_service.GetUserByID(userID)
	if errors.Is(err, user_service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("查询当前用户失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户信息失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    user,
	})
}
