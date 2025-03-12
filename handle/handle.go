package handle

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strings"
	"yuklovchiBot/admin"
	"yuklovchiBot/pkg/state"
	"yuklovchiBot/storage"
)

func HandleUpdate(update tgbotapi.Update, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	if update.Message != nil {

		handleMessage(update.Message, db, botInstance)
	} else if update.CallbackQuery != nil {
		// Callback query'ni qayta ishlash
		handleCallbackQuery(update.CallbackQuery, db, botInstance)
	} else {
		log.Printf("Unsupported update type: %T", update)
	}
}

func handleMessage(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	log.Printf("Received message: %s", text)

	if userState, exists := state.UserStates[chatID]; exists {
		log.Printf("User state: %s", userState)
		switch userState {
		case "waiting_for_broadcast_message":
			admin.HandleBroadcastMessage(msg, db, botInstance)
			delete(state.UserStates, chatID)
			return
		case "waiting_for_channel_link":
			admin.HandleChannelLink(msg, db, botInstance)
			delete(state.UserStates, chatID)
			return
		case "waiting_for_admin_id":
			admin.HandleAdminAdd(msg, db, botInstance)
			delete(state.UserStates, chatID)
			return
		case "waiting_for_admin_id_remove":
			admin.HandleAdminRemove(msg, db, botInstance)
			delete(state.UserStates, chatID)
			return
		}
	}

	if text == "/start" {
		handleStartCommand(msg, db, botInstance)
		err := storage.AddUserToDatabase(db, int(msg.Chat.ID))
		if err != nil {
			log.Printf("Error adding user to database: %v", err)
		}
	} else if text == "/admin" {
		admin.HandleAdminCommand(msg, db, botInstance)
	} else {
		handleDefaultMessage(msg, db, botInstance)
	}
}

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID
	data := callbackQuery.Data

	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		return
	}

	switch {
	// 1) Foydalanuvchi obunani tekshirish
	case callbackQuery.Data == "check_subscription":
		if isUserSubscribedToChannels(chatID, channels, botInstance) {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			_, err := botInstance.Send(deleteMsg)
			if err != nil {
				log.Printf("Error sending newmessage: %v", err)
				return
			}

			welcomeMessage := fmt.Sprintf("👋 Assalomu alaykum [%s](tg://user?id=%d) botimizga xush kelibsiz.", callbackQuery.From.FirstName, callbackQuery.From.ID)
			msg := tgbotapi.NewMessage(chatID, welcomeMessage)
			msg.ParseMode = "Markdown"
			_, err = botInstance.Send(msg)
			if err != nil {
				log.Printf("Error sending photo: %v", err)
				return
			}
		} else {
			msg := tgbotapi.NewMessage(chatID, "Iltimos, kanallarga azo bo'ling.")
			inlineKeyboard := createSubscriptionKeyboard(channels)
			msg.ReplyMarkup = inlineKeyboard
			botInstance.Send(msg)
		}

	// 2) Kanalni o‘chirishga doir callback
	case strings.HasPrefix(callbackQuery.Data, "delete_channel_"):
		channel := strings.TrimPrefix(callbackQuery.Data, "delete_channel_")
		admin.AskForChannelDeletionConfirmation(chatID, messageID, channel, botInstance)
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		botInstance.Send(deleteMsg)

	case strings.HasPrefix(callbackQuery.Data, "confirm_delete_channel_"):
		channel := strings.TrimPrefix(callbackQuery.Data, "confirm_delete_channel_")
		admin.DeleteChannel(chatID, messageID, channel, db, botInstance)
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		botInstance.Send(deleteMsg)

	case callbackQuery.Data == "cancel_delete_channel":
		admin.CancelChannelDeletion(chatID, messageID, botInstance)

	// 3) **Yangi qo‘shilgan: Instagram audio yuklash callback**
	case strings.HasPrefix(data, "download_insta_audio|"):
		parts := strings.SplitN(data, "|", 2)
		if len(parts) == 2 {
			videoFile := parts[1]
			downloadAndSendInstaAudio(chatID, videoFile, botInstance)

			err := os.Remove(videoFile)
			if err != nil {
				log.Printf("Xatolik: Video faylni o‘chirishda xatolik: %v", err)
			} else {
				log.Printf("Fayl o‘chirildi: %s", videoFile)
			}
		}
		RemoveInlineKeyboardAndUpdateCaption(chatID, botInstance)

	// 🎯 Agar foydalanuvchi "Yo‘q" bosgan bo‘lsa, videoni o‘chirib tashlaymiz
	case strings.HasPrefix(data, "skip_insta_audio|"):
		parts := strings.SplitN(data, "|", 2)
		if len(parts) == 2 {
			videoFile := parts[1]

			// 📌 Videoni o‘chiramiz (faqat serverdan)
			err := os.Remove(videoFile)
			if err != nil {
				log.Printf("Xatolik: Video faylni o‘chirishda xatolik: %v", err)
			} else {
				log.Printf("Fayl o‘chirildi: %s", videoFile)
			}
		}
		RemoveInlineKeyboardAndUpdateCaption(chatID, botInstance) // ✅ Tugmalarni o‘chirish va captionni yangilash

	// 4) Xuddi shu uslubda TikTok audio yuklash callback’lari ham qo‘shishingiz mumkin
	case strings.HasPrefix(data, "download_tiktok_audio|"):
		parts := strings.SplitN(data, "|", 2)
		if len(parts) == 2 {
			videoFile := parts[1]
			downloadAndSendTikTokAudio(chatID, videoFile, botInstance)

			// ✅ Audio yuborilgandan keyin videoni o‘chirib tashlaymiz
			err := os.Remove(videoFile)
			if err != nil {
				log.Printf("Xatolik: Video faylni o‘chirishda xatolik: %v", err)
			} else {
				log.Printf("Fayl o‘chirildi: %s", videoFile)
			}
		}
		RemoveInlineKeyboardAndUpdateCaption(chatID, botInstance)

	// 🎯 Agar foydalanuvchi "Yo‘q" bosgan bo‘lsa, videoni **lokaldan o‘chirib tashlaymiz**
	case strings.HasPrefix(data, "skip_tiktok_audio|"):
		parts := strings.SplitN(data, "|", 2)
		if len(parts) == 2 {
			videoFile := parts[1]
			err := os.Remove(videoFile)
			if err != nil {
				log.Printf("Xatolik: Video faylni o‘chirishda xatolik: %v", err)
			} else {
				log.Printf("Fayl o‘chirildi: %s", videoFile)
			}
		}
		RemoveInlineKeyboardAndUpdateCaption(chatID, botInstance)

	case strings.HasPrefix(data, "youtube_download|"):
		HandleYouTubeDownloadCallback(chatID, messageID, data, botInstance)

	default:
		log.Printf("Unknown callback data: %s", callbackQuery.Data)
	}
}
func handleStartCommand(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	firstName := msg.From.FirstName

	log.Printf("Adding user to database: %d ", userID)
	err := storage.AddUserToDatabase(db, userID)
	if err != nil {
		log.Printf("Error adding user to database: %v", err)
		return
	}

	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		return
	}

	if isUserSubscribedToChannels(chatID, channels, botInstance) {
		welcomeMessage := fmt.Sprintf("👋 Assalomu alaykum [%s](tg://user?id=%d), botimizga xush kelibsiz.\n\nMen sizga Instagram va TikTokdan videolarni yuklashda yordam beruvchi botman.\n\n Iltimos menga video havolasini yuboring.", firstName, userID)

		msg := tgbotapi.NewMessage(chatID, welcomeMessage)
		msg.ParseMode = "Markdown"
		_, err := botInstance.Send(msg)
		if err != nil {
			log.Printf("Error sending welcome message: %v", err)
			return
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, "Iltimos, kanallarga azo bo'ling.")
		inlineKeyboard := createSubscriptionKeyboard(channels)
		msg.ReplyMarkup = inlineKeyboard
		botInstance.Send(msg)
	}
}

func handleDefaultMessage(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	if strings.HasPrefix(text, "https://www.instagram.com/") || strings.HasPrefix(text, "instagram") {
		loadingMsg, err := botInstance.Send(tgbotapi.NewMessage(chatID, "⌛️"))
		if err != nil {
			log.Printf("Loading xabarini yuborishda xatolik: %v", err)
		}

		downloadAndSendInstaVideo(chatID, text, botInstance, loadingMsg.MessageID)
		return
	}

	if strings.HasPrefix(text, "https://www.tiktok.com/") || strings.HasPrefix(text, "tiktok") {

		loadingMsg, err := botInstance.Send(tgbotapi.NewMessage(chatID, "⌛️"))
		if err != nil {
			log.Printf("Loading xabarini yuborishda xatolik: %v", err)
		}

		downloadAndSendTikTokVideo(chatID, text, botInstance, loadingMsg.MessageID)
		return
	}

	switch text {
	case "Kanal qo'shish":
		state.UserStates[chatID] = "waiting_for_channel_link"
		msgResponse := tgbotapi.NewMessage(chatID, "Kanal linkini yuboring (masalan, https://t.me/your_channel):")
		botInstance.Send(msgResponse)
	case "Admin qo'shish":
		state.UserStates[chatID] = "waiting_for_admin_id"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, yangi admin ID sini yuboring:")
		botInstance.Send(msgResponse)
	case "Admin o'chirish":
		state.UserStates[chatID] = "waiting_for_admin_id_remove"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, admin ID sini o'chirish uchun yuboring:")
		botInstance.Send(msgResponse)
	case "Kanal o'chirish":
		admin.DisplayChannelsForDeletion(chatID, db, botInstance)
	case "Statistika":
		admin.HandleStatistics(msg, db, botInstance)
	case "Habar yuborish":
		state.UserStates[chatID] = "waiting_for_broadcast_message"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, yubormoqchi bo'lgan habaringizni kiriting (Bekor qilish uchun /cancel):")
		botInstance.Send(msgResponse)
	case "BackUp olish":
		if storage.IsAdmin(int(chatID), db) {
			go HandleBackup(db, botInstance)
		}
	}
}

func isUserSubscribedToChannels(chatID int64, channels []string, botInstance *tgbotapi.BotAPI) bool {
	for _, channel := range channels {
		log.Printf("Checking subscription to channel: %s", channel)
		chat, err := botInstance.GetChat(tgbotapi.ChatConfig{SuperGroupUsername: "@" + channel})
		if err != nil {
			log.Printf("Error getting chat info for channel %s: %v", channel, err)
			return false
		}

		member, err := botInstance.GetChatMember(tgbotapi.ChatConfigWithUser{
			ChatID: chat.ID,
			UserID: int(chatID),
		})
		if err != nil {
			log.Printf("Error getting chat member info for channel %s: %v", channel, err)
			return false
		}
		if member.Status == "left" || member.Status == "kicked" {
			log.Printf("User %d is not subscribed to channel %s", chatID, channel)
			return false
		}
	}
	return true
}

func createSubscriptionKeyboard(channels []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, channel := range channels {
		channelName := strings.TrimPrefix(channel, "https://t.me/")
		button := tgbotapi.NewInlineKeyboardButtonURL("Kanalga azo bo'lish", "https://t.me/"+channelName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}
	checkButton := tgbotapi.NewInlineKeyboardButtonData("Azo bo'ldim", "check_subscription")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(checkButton))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func RemoveInlineKeyboardAndUpdateCaption(chatID int64, botInstance *tgbotapi.BotAPI) {
	// Xabar ID'sini olish
	messageID, exists := state.GetMessageID(chatID)
	if !exists {
		log.Printf("Xatolik: Chat %d uchun xabar topilmadi", chatID)
		return
	}

	// 📌 Xabar captionini faqat "Siz so‘ragan video." qilib yangilash
	editMsg := tgbotapi.NewEditMessageCaption(chatID, messageID, "Siz so‘ragan video.")
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = nil // 📌 Inline tugmalarni olib tashlaymiz

	_, err := botInstance.Send(editMsg)
	if err != nil {
		log.Printf("Xabarni yangilashda xatolik: %v", err)
	}
}

// 🎯 "Ha" va "Yo‘q" tugmalarini yaratish
func createAudioOptionKeyboard(platform, videoPath string) tgbotapi.InlineKeyboardMarkup {
	haData := fmt.Sprintf("download_%s_audio|%s", platform, videoPath)
	yoqData := fmt.Sprintf("skip_%s_audio|%s", platform, videoPath)

	haButton := tgbotapi.NewInlineKeyboardButtonData("Ha", haData)
	yoqButton := tgbotapi.NewInlineKeyboardButtonData("Yo‘q", yoqData)

	row := tgbotapi.NewInlineKeyboardRow(haButton, yoqButton)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}
