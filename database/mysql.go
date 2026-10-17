package database

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var DB *gorm.DB

func InitDB() error {
	dsn := "root:root@tcp(127.0.0.1:3306)/go_douyin?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接 MYSQL 失败：%w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return fmt.Errorf("MySQL Ping 失败: %w", err)
	}

	DB = db
	return nil
}
func CloseMySQL() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
