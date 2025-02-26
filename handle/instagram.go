package handle

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"net/http"
	"yuklovchiBot/config"
)

type VideoResponse struct {
	Status string `json:"status"`
	Data   struct {
		Filename string `json:"filename"`
		VideoURL string `json:"videoUrl"`
	} `json:"data"`
}

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

	// API javobini JSON formatida o'qish
	var videoResp VideoResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Javobni o'qishda xatolik yuz berdi."))
		return
	}

	err = json.Unmarshal(body, &videoResp)
	if err != nil || videoResp.Status != "success" {
		botInstance.Send(tgbotapi.NewMessage(chatID, "❌ Video yuklab olishda muammo bor."))
		return
	}

	// Video faylni foydalanuvchiga yuborish
	videoMsg := tgbotapi.NewVideoShare(chatID, videoResp.Data.VideoURL)
	botInstance.Send(videoMsg)
}
