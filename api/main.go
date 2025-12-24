package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/berduk-dev/bad-da-yo/internal/handler"
	"github.com/berduk-dev/bad-da-yo/internal/repo"
	"github.com/berduk-dev/bad-da-yo/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Получаем DATABASE_URL из переменных окружения
	dbURL := os.Getenv("DATABASE_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	defer pool.Close()

	// Проверим соединение
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Ошибка ping:", err)
	}

	r := gin.Default()

	bdyRepository := repo.New(pool)
	bdyService := service.New(bdyRepository)
	bdyHandler := handler.New(bdyService)

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_URL")}, // разрешённые домены
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/prize", bdyHandler.CreatePrize) // получить приз + отправить код

	_ = r.Run(":8080")
}
