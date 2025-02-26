package admin

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
	"yuklovchiBot/models"
	"yuklovchiBot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func HandleAdminCommand(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		msgResponse := tgbotapi.NewMessage(chatID, "Siz admin emassiz.")
		botInstance.Send(msgResponse)
		return
	}

	adminKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Statistika"),
			tgbotapi.NewKeyboardButton("Habar yuborish"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Kanal qo'shish"),
			tgbotapi.NewKeyboardButton("Kanal o'chirish"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Admin qo'shish"),
			tgbotapi.NewKeyboardButton("Admin o'chirish"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("BackUp olish"),
		),
	)

	msgResponse := tgbotapi.NewMessage(chatID, "Admin buyrug'lari:")
	msgResponse.ReplyMarkup = adminKeyboard
	botInstance.Send(msgResponse)
}

func HandleChannelLink(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {

	chatID := msg.Chat.ID
	channelLink := msg.Text

	if !storage.IsAdmin(int(chatID), db) {
		msgResponse := tgbotapi.NewMessage(chatID, "Siz admin emassiz.")
		botInstance.Send(msgResponse)
		return
	}

	err := storage.AddChannelToDatabase(db, channelLink)
	if err != nil {
		log.Printf("Error adding channel to database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanalni qo'shishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Kanal muvaffaqiyatli qo'shildi.")
	botInstance.Send(msgResponse)
}

func DeleteChannel(chatID int64, messageID int, channel string, db *sql.DB, botInstance *tgbotapi.BotAPI) {

	if !storage.IsAdmin(int(chatID), db) {
		return
	}

	err := storage.DeleteChannelFromDatabase(db, channel)
	if err != nil {
		log.Printf("Error deleting channel from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanalni o'chirishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s kanali muvaffaqiyatli o'chirildi.", channel))
	botInstance.Send(msgResponse)
}

func CancelChannelDeletion(chatID int64, messageID int, botInstance *tgbotapi.BotAPI) {

	msgResponse := tgbotapi.NewMessage(chatID, "Kanal o'chirish bekor qilindi.")
	botInstance.Send(msgResponse)

	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)
}

func HandleAdminAdd(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {

	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		return
	}

	adminID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Noto'g'ri admin ID formati.")
		botInstance.Send(msgResponse)
		return
	}

	err = storage.AddAdminToDatabase(db, adminID)
	if err != nil {
		log.Printf("Error adding admin to database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Admin qo'shishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Admin muvaffaqiyatli qo'shildi.")
	botInstance.Send(msgResponse)
}

func HandleAdminRemove(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		return
	}

	adminID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Noto'g'ri admin ID formati.")
		botInstance.Send(msgResponse)
		return
	}

	err = storage.RemoveAdminFromDatabase(db, adminID)
	if err != nil {
		log.Printf("Error removing admin from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Admin o'chirishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Admin muvaffaqiyatli o'chirildi.")
	botInstance.Send(msgResponse)
}

func DisplayChannelsForDeletion(chatID int64, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanallarni olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, channel := range channels {
		button := tgbotapi.NewInlineKeyboardButtonData(channel, "delete_channel_"+channel)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msgResponse := tgbotapi.NewMessage(chatID, "O'chirilishi kerak bo'lgan kanalni tanlang:")
	msgResponse.ReplyMarkup = inlineKeyboard
	botInstance.Send(msgResponse)
}

func AskForChannelDeletionConfirmation(chatID int64, messageID int, channel string, botInstance *tgbotapi.BotAPI) {
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("Ha", "confirm_delete_channel_"+channel)
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("Yo'q", "cancel_delete_channel")

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(confirmButton, cancelButton),
	)
	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s kanalini o'chirmoqchimisiz?", channel))
	msgResponse.ReplyMarkup = inlineKeyboard
	botInstance.Send(msgResponse)

	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)
}

func HandleStatistics(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		return
	}

	// Fetch user statistics from the database
	totalUsers, err := storage.GetTotalUsers(db)
	if err != nil {
		log.Printf("Error getting total users: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Statistikani olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	todayUsers, err := storage.GetTodayUsers(db)
	if err != nil {
		log.Printf("Error getting today's users: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Statistikani olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	lastMonthUsers, err := storage.GetLastMonthUsers(db)
	if err != nil {
		log.Printf("Error getting last month's users: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Statistikani olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	// Create the response message
	statsMessage := fmt.Sprintf(
		"Foydalanuvchilar statistikasi:\n\nBugun qo'shilgan foydalanuvchilar: %d\nOxirgi 1 oy ichida qo'shilgan foydalanuvchilar: %d\nUmumiy foydalanuvchilar soni: %d",
		todayUsers, lastMonthUsers, totalUsers,
	)

	msgResponse := tgbotapi.NewMessage(chatID, statsMessage)
	botInstance.Send(msgResponse)
}

func HandleBroadcastMessage(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		return
	}

	if msg.Text == "/cancel" {
		msgResponse := tgbotapi.NewMessage(chatID, "Habar yuborish bekor qilindi.")
		botInstance.Send(msgResponse)
		return
	}

	users, err := storage.GetAllUsers(db)
	if err != nil {
		log.Printf("Error retrieving users: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Foydalanuvchilarni olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	go sendBroadcastMessage(users, msg.Text, chatID, botInstance)
	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("Habar %d foydalanuvchilarga yuborilmoqda...", len(users)))
	botInstance.Send(msgResponse)
}

func sendBroadcastMessage(users []models.User, message string, adminChatID int64, botInstance *tgbotapi.BotAPI) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	count := 0
	for _, user := range users {
		<-ticker.C
		msg := tgbotapi.NewMessage(user.ID, message)
		if _, err := botInstance.Send(msg); err != nil {
			log.Printf("Error sending message to user %d: %v", user.ID, err)
		} else {
			count++
		}
	}

	log.Printf("Broadcast completed. Sent %d messages.", count)
	msgResponse := tgbotapi.NewMessage(adminChatID, fmt.Sprintf("BroadcasadminChatIDt completed. Sent %d messages.", count))
	botInstance.Send(msgResponse)
}
