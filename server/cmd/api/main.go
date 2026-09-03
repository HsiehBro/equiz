package main

import (
	"fmt"
	"log"

	"exam-server/internal/config"
	"exam-server/internal/database"
	"exam-server/internal/router"
)

func main() {
	log.Println("==================================================")
	log.Println("         智能刷题助手 API 后端服务启动中...         ")
	log.Println("==================================================")

	// 1. 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 2. 初始化 PostgreSQL 数据库连接与表结构自动迁移
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败 (请检查虚拟机 PostgreSQL 配置): %v", err)
	}

	// 3. 构建路由
	r := router.SetupRouter(db, cfg)

	// 4. 监听启动
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("服务启动成功，正在监听 http://0.0.0.0%s\n", addr)
	log.Println("探活检查地址: http://127.0.0.1" + addr + "/health")
	log.Println("Swagger / API 路由前缀: /api/v1")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务运行异常: %v", err)
	}
}
