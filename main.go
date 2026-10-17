package main

import (
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/routers"
	"log"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	// 1. 初始化 MySQL
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}

	// 2. 程序退出时关闭数据库连接
	defer func() {
		if err := database.CloseMySQL(); err != nil {
			log.Printf("关闭 MySQL 失败: %v", err)
		}
	}()
	// 3. 创建数据库表
	if err := database.DB.AutoMigrate(&models.User{}, &models.Video{},
		&models.Comment{}, &models.VideoLike{},
		&models.Follow{}); err != nil {
		log.Fatal("数据库迁移失败: ", err)
	}
	log.Println("MySQL 数据库初始化成功")
	log.Println("五张业务表迁移完成")
	r := routers.InitRouter()
	r.Run(":8080")

}
