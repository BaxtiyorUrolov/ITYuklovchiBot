package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"yuklovchiBot/config"
	"yuklovchiBot/handle"
	"yuklovchiBot/pkg/logger"
	"yuklovchiBot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Environment, "IT-Yuklovchi-Bot")
	botToken := cfg.BotToken

	db, err := storage.New(context.Background(), cfg, log)
	if err != nil {
		log.Error("error while connecting database", logger.Error(err))
		return
	}

	botInstance, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Error("Failed to create Telegram bot", logger.Error(err))
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start Telegram bot updates
	go startTelegramBot(ctx, db, botInstance)

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info("Shutdown signal received")

}

func startTelegramBot(ctx context.Context, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	offset := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Telegram bot...")
			return
		default:
			updates, err := botInstance.GetUpdates(tgbotapi.NewUpdate(offset))
			if err != nil {
				log.Printf("Error getting updates: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updates {
				handle.HandleUpdate(update, db, botInstance)
				offset = update.UpdateID + 1
			}
		}
	}
}
