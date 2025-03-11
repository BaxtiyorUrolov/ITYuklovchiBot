package handle

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
	"yuklovchiBot/pkg/state"
)

// TikTok API javob strukturasini e'lon qilamiz
type TikTokResponse struct {
	Data struct {
		Play string `json:"play"`
	} `json:"data"`
}

// extractVideoID - TikTok video URL'sidan video ID ni ajratib oladi
func extractVideoID(url string) string {
	re := regexp.MustCompile(`(?:https?:\/\/)?(?:www\.)?tiktok\.com\/(?:.*\/)?([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return "video"
}

// downloadTikTokVideo - berilgan TikTok video linki bo'yicha videoni yuklab olib,
// "videos" papkasiga saqlaydi va fayl yo'lini qaytaradi.
func downloadTikTokVideo(videoLink string) (string, error) {
	// Video ID ni ajratish
	videoID := extractVideoID(videoLink)
	if videoID == "video" {
		return "", fmt.Errorf("invalid video URL")
	}

	// API URL'ni olish uchun Base64 kodlangan satrni dekodlash
	base64URL := "aHR0cHM6Ly90aWt3bS5jb20vYXBpLw=="
	decodedURLBytes, err := base64.StdEncoding.DecodeString(base64URL)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 URL: %w", err)
	}
	decodedURL := string(decodedURLBytes)

	client := &http.Client{Timeout: 120 * time.Second}

	// API ga soʻrov yuborish video maʼlumotlarini olish uchun
	apiURL := fmt.Sprintf("%s?url=%s", decodedURL, videoLink)
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("error fetching video info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching video info: received status code %d", resp.StatusCode)
	}

	var videoResp TikTokResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if videoResp.Data.Play == "" {
		return "", fmt.Errorf("video URL not found in the response")
	}

	// Video URL orqali video faylini yuklab olish
	videoDownloadURL := videoResp.Data.Play
	downloadResp, err := client.Get(videoDownloadURL)
	if err != nil {
		return "", fmt.Errorf("error downloading video: %w", err)
	}
	defer downloadResp.Body.Close()

	// "videos" papkasini mavjudligini tekshirish va agar mavjud bo'lmasa yaratish
	if _, err := os.Stat("videos"); os.IsNotExist(err) {
		if err = os.Mkdir("videos", os.ModePerm); err != nil {
			return "", fmt.Errorf("error creating videos folder: %w", err)
		}
	}

	// Video faylini saqlash uchun fayl nomini yaratish (masalan, videoID.mp4)
	filePath := fmt.Sprintf("videos/%s.mp4", videoID)
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creating video file: %w", err)
	}
	defer outFile.Close()

	// Video faylini diskka yozish
	_, err = io.Copy(outFile, downloadResp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving video file: %w", err)
	}

	return filePath, nil
}

// 📌 TikTok videoni yuklab olish va foydalanuvchiga yuborish
func downloadAndSendTikTokVideo(chatID int64, videoURL string, botInstance *tgbotapi.BotAPI) {
	// TikTok video faylini yuklab olib, lokal yo'lni olamiz.
	filePath, err := downloadTikTokVideo(videoURL)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Video yuklab olishda xatolik yuz berdi: %s", err)))
		return
	}

	// Video yuborish uchun Telegram video xabarini yaratamiz.
	videoMsg := tgbotapi.NewVideoUpload(chatID, filePath)
	videoMsg.Caption = "Siz so‘ragan video.\n\nAudiosini yuklashni istaysizmi?"
	// createAudioOptionKeyboard - foydalanuvchiga audio yuklash opsiyasini taklif qiluvchi tugmalarni yaratadi.
	videoMsg.ReplyMarkup = createAudioOptionKeyboard("tiktok", filePath)

	sentMsg, err := botInstance.Send(videoMsg)
	if err != nil {
		log.Printf("Video yuborishda xatolik: %v", err)
		return
	}

	// Xabar ID'sini saqlaymiz (keyinchalik tugmalarni o‘chirish uchun)
	state.SaveMessageID(chatID, sentMsg.MessageID)

	// Logga video fayl yo'lini chiqarish (agar kerak bo'lsa)
	log.Printf("Video muvaffaqiyatli yuklab olindi: %s", filePath)
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
