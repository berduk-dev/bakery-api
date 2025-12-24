package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"tgbot-bad-da-yo/internal/handler"
	"tgbot-bad-da-yo/internal/repo"
	"tgbot-bad-da-yo/internal/service"
)

func main() {
	_ = godotenv.Load("../.env")

	// Получаем DATABASE_URL из переменных окружения
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback для локальной разработки
		dbURL = "postgres://badayo:badayo123@localhost:5432/badayo?sslmode=disable"
	}

	fmt.Println("🔗 Подключение к PostgreSQL...")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer pool.Close()

	// Проверим соединение
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Ошибка ping БД:", err)
	}

	fmt.Println("🚀 Подключение к базе данных успешно установлено.")

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	developerID, _ := strconv.ParseInt(os.Getenv("DEVELOPER_TG_ID"), 10, 64)
	adminChatID, _ := strconv.ParseInt(os.Getenv("ADMIN_TELEGRAM_CHAT_ID"), 10, 64)

	r := repo.New(pool)
	s := service.New(r, bot)
	h := handler.New(bot, s, adminID, developerID, adminChatID)

	h.Start()
}
