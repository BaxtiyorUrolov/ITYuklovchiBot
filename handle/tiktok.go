package handle

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"log"
	"net/http"
	"os"
	"yuklovchiBot/config"
	"yuklovchiBot/pkg/state"
)

// TikTok API javob strukturasini e'lon qilamiz
type TikTokResponse struct {
	DownloadURL string `json:"download_url"`
}

// 📌 TikTok videoni yuklab olish va foydalanuvchiga yuborish
func downloadAndSendTikTokVideo(chatID int64, videoURL string, botInstance *tgbotapi.BotAPI) {
	tikTokApi := config.Load().TikTokApi

	// API'ga so‘rov yuborish
	apiURL := fmt.Sprintf("%s%s", tikTokApi, videoURL)
	resp, err := http.Get(apiURL)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda xatolik yuz berdi."))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olinmadi. Iltimos, boshqa linkni sinab ko'ring."))
		return
	}

	// API javobini JSON formatida o‘qish
	var tiktokResp TikTokResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ API javobini o‘qishda xatolik yuz berdi."))
		return
	}

	err = json.Unmarshal(body, &tiktokResp)
	if err != nil || tiktokResp.DownloadURL == "" {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda muammo bor."))
		return
	}

	// 📌 Videoni **lokalga yuklab olamiz**
	videoFile, err := downloadFile(tiktokResp.DownloadURL, "temp_tiktok_", ".mp4")
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda xatolik."))
		return
	}

	// 📌 Videoni foydalanuvchiga yuborish
	videoMsg := tgbotapi.NewVideoUpload(chatID, videoFile)
	videoMsg.Caption = "Siz so‘ragan video.\n\nAudiosini yuklashni istaysizmi?"
	videoMsg.ReplyMarkup = createAudioOptionKeyboard("tiktok", videoFile)

	sentMsg, err := botInstance.Send(videoMsg)
	if err != nil {
		log.Printf("Video yuborishda xatolik: %v", err)
	}

	// 📌 Xabar ID'ni saqlash (tugmalarni o‘chirish uchun)
	state.SaveMessageID(chatID, sentMsg.MessageID)
}

// 📌 TikTok videodan audio ajratish va foydalanuvchiga yuborish
func downloadAndSendTikTokAudio(chatID int64, videoFile string, botInstance *tgbotapi.BotAPI) {
	audioFile, err := extractAudio(videoFile)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Audio ajratishda xatolik yuz berdi."))
		return
	}
	defer os.Remove(audioFile)

	// Audio faylni foydalanuvchiga yuborish
	audioMsg := tgbotapi.NewAudioUpload(chatID, audioFile)
	audioMsg.Caption = "Mana videoning audio fayli:"
	if _, err := botInstance.Send(audioMsg); err != nil {
		log.Printf("Audio yuborishda xatolik: %v", err)
	}
}
