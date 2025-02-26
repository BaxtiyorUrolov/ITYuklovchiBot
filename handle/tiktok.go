package handle

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"net/http"
	"yuklovchiBot/config"
)

// TikTok API javob strukturasini e'lon qilamiz
type TikTokResponse struct {
	DownloadURL string `json:"download_url"`
}

// TikTok videoni yuklab olish va yuborish funksiyasi
func downloadAndSendTikTokVideo(chatID int64, videoURL string, botInstance *tgbotapi.BotAPI) {

	tikTokApi := config.Load().TikTokApi

	// TikTok video yuklash API'siga so‘rov yuborish
	apiURL := fmt.Sprintf("%s%s", tikTokApi, videoURL)
	resp, err := http.Get(apiURL)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda xatolik yuz berdi."))
		return
	}
	defer resp.Body.Close()

	// API'dan noto‘g‘ri javob kelsa
	if resp.StatusCode != http.StatusOK {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olinmadi. Iltimos, boshqa linkni sinab ko'ring."))
		return
	}

	// API javobini JSON formatida o'qish
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

	// Video havolani foydalanuvchiga jo‘natish
	videoMsg := tgbotapi.NewVideoShare(chatID, tiktokResp.DownloadURL)
	botInstance.Send(videoMsg)
}
