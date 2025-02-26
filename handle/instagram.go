package handle

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
	"yuklovchiBot/config"
	"yuklovchiBot/pkg/state"
)

// API'dan qaytgan video javob formati
type VideoResponse struct {
	Status string `json:"status"`
	Data   struct {
		Filename string `json:"filename"`
		VideoURL string `json:"videoUrl"`
	} `json:"data"`
}

// 📌 1️⃣ Videoni yuklab, keyin foydalanuvchiga yuborish
func downloadAndSendInstaVideo(chatID int64, videoURL string, botInstance *tgbotapi.BotAPI) {
	instaApi := config.Load().InstaApi

	// API'ga so‘rov yuborish
	apiURL := fmt.Sprintf("%s%s", instaApi, videoURL)
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
	var videoResp VideoResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ API javobini o‘qishda xatolik yuz berdi."))
		return
	}

	err = json.Unmarshal(body, &videoResp)
	if err != nil || videoResp.Status != "success" {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda muammo bor."))
		return
	}

	// 📌 2️⃣ Videoni **lokalga** yuklab olamiz
	videoFile, err := downloadFile(videoResp.Data.VideoURL, "temp_insta_", ".mp4")
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda xatolik."))
		return
	}

	// 📌 3️⃣ Videoni foydalanuvchiga yuborish
	videoMsg := tgbotapi.NewVideoUpload(chatID, videoFile)
	videoMsg.Caption = "Siz so‘ragan video.\n\nAudiosini yuklashni istaysizmi?"
	videoMsg.ReplyMarkup = createAudioOptionKeyboard("insta", videoFile)

	sentMsg, err := botInstance.Send(videoMsg)
	if err != nil {
		log.Printf("Video yuborishda xatolik: %v", err)
	}

	// Xabar ID'sini saqlaymiz (keyinchalik tugmani o‘chirish uchun)
	messageID := sentMsg.MessageID
	state.SaveMessageID(chatID, messageID)
}

// 📌 6️⃣ Inline tugmalarni olib tashlash (xabarni tahrirlash)
func removeInlineKeyboard(chatID int64, messageID int, botInstance *tgbotapi.BotAPI) {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{})
	botInstance.Send(edit)
}

// 📌 7️⃣ Yuklab olish funksiyasi
func downloadFile(fileURL, prefix, ext string) (string, error) {
	fileName := fmt.Sprintf("%s%d%s", prefix, time.Now().UnixNano(), ext)
	out, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

// 📌 8️⃣ ffmpeg yordamida audioni ajratish (audio.mp3 nomi bilan)
func extractAudio(videoFile string) (string, error) {
	audioFile := "audio.mp3" // 🎯 Faqat "audio.mp3" nomi bilan ajratamiz

	cmd := exec.Command("ffmpeg", "-i", videoFile, "-vn", "-acodec", "libmp3lame", "-y", audioFile)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return audioFile, nil
}

// 📌 9️⃣ Audioni foydalanuvchiga yuborish
func downloadAndSendInstaAudio(chatID int64, videoFile string, botInstance *tgbotapi.BotAPI) {
	// ffmpeg yordamida audio ajratamiz
	audioFile, err := extractAudio(videoFile)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Audio ajratishda xatolik yuz berdi."))
		return
	}
	defer os.Remove(audioFile) // 🎯 Audio faylni yuborgach o‘chirib tashlaymiz

	// Audio faylni foydalanuvchiga yuborish
	audioMsg := tgbotapi.NewAudioUpload(chatID, audioFile)
	audioMsg.Caption = "Mana videoning audio fayli:"
	if _, err := botInstance.Send(audioMsg); err != nil {
		log.Printf("Audio yuborishda xatolik: %v", err)
	}
}
