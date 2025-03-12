package handle

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"yuklovchiBot/pkg/state"
)

type YouTubeMetadata struct {
	Title    string          `json:"title"`
	Duration float64         `json:"duration"`
	Formats  []YouTubeFormat `json:"formats"`
}

type YouTubeFormat struct {
	FormatID       string  `json:"format_id"`
	FormatNote     string  `json:"format_note"`
	Ext            string  `json:"ext"`
	Filesize       float64 `json:"filesize"`
	FilesizeApprox float64 `json:"filesize_approx"`
	Height         int     `json:"height"`
	Acodec         string  `json:"acodec"`
	Vcodec         string  `json:"vcodec"`
}

// Video havolasi va metadata’ni saqlab turish uchun
var YouTubeVideoLinkCache = make(map[int64]string)
var YouTubeVideoInfo = make(map[int64]YouTubeMetadata)

// Foydalanuvchi YouTube link yuborganda chaqiriladigan asosiy funksiya
func HandleYouTubeLink(chatID int64, videoURL string, bot *tgbotapi.BotAPI) error {
	// 0) Linkni cache’da saqlaymiz
	YouTubeVideoLinkCache[chatID] = videoURL

	// 1) `yt-dlp --dump-json` orqali metadata olish
	cmd := exec.Command("yt-dlp", "--dump-json", videoURL)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("yt-dlp bilan metadata olishda xatolik: %v", err)
	}

	// 2) JSON’ni parse qilish
	var meta YouTubeMetadata
	if err := json.Unmarshal(output, &meta); err != nil {
		return fmt.Errorf("JSON parse xatosi: %v", err)
	}
	// Keshga saqlaymiz
	YouTubeVideoInfo[chatID] = meta

	// 3) 360p, 480p, 720p, 1080p rezlar orasidan eng kattasi + eng yaxshi audio
	largestByRes, bestAudio := filterLargestFormats(meta)

	// 4) InlineKeyboard tayyorlash
	kb := buildInlineKeyboardForLargestFormats(largestByRes, bestAudio)

	// 5) Xabarni yuborish
	durStr := formatDuration(meta.Duration)
	caption := fmt.Sprintf("*%s*\nDuration: %s\nChoose format to download:", meta.Title, durStr)

	msg := tgbotapi.NewMessage(chatID, caption)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb

	sentMsg, err := bot.Send(msg)
	if err != nil {
		return err
	}

	// Keyin tugmalarni olib tashlash uchun xabar ID sini saqlaymiz
	state.SaveMessageID(chatID, sentMsg.MessageID)
	return nil
}

// filterLargestFormats: har bir (360, 480, 720, 1080)p uchun eng katta faylni va eng yaxshi audio’ni tanlaydi
func filterLargestFormats(meta YouTubeMetadata) (map[int]YouTubeFormat, *YouTubeFormat) {
	desiredResolutions := []int{360, 480, 720, 1080}
	largestByRes := make(map[int]YouTubeFormat)
	var bestAudio *YouTubeFormat

	for _, f := range meta.Formats {
		// real hajmni (filesizeApprox) hisobga olamiz
		sizeBytes := f.Filesize
		if sizeBytes == 0 && f.FilesizeApprox > 0 {
			sizeBytes = f.FilesizeApprox
		}

		// Agar video codeci 'none' bo‘lsa, demak audio-only format
		isAudio := (f.Vcodec == "none")
		if isAudio {
			// bestAudio topish
			if bestAudio == nil || sizeBytes > bestAudio.Filesize {
				copyF := f
				copyF.Filesize = sizeBytes
				bestAudio = &copyF
			}
		} else {
			// MP4 format va balandligi (Height) bor bo‘lsa
			if f.Ext == "mp4" && f.Height > 0 {
				for _, res := range desiredResolutions {
					if f.Height == res {
						current, ok := largestByRes[res]
						if !ok || sizeBytes > current.Filesize {
							copyF := f
							copyF.Filesize = sizeBytes
							largestByRes[res] = copyF
						}
					}
				}
			}
		}
	}

	return largestByRes, bestAudio
}

// buildInlineKeyboardForLargestFormats: topilgan formatlar uchun tugmalar yaratadi
func buildInlineKeyboardForLargestFormats(largestByRes map[int]YouTubeFormat, bestAudio *YouTubeFormat) tgbotapi.InlineKeyboardMarkup {
	sortedRes := []int{360, 480, 720, 1080}
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, res := range sortedRes {
		f, ok := largestByRes[res]
		if !ok {
			continue
		}
		sizeMB := f.Filesize / 1024 / 1024
		btnText := fmt.Sprintf("%dp - %.1fMB", res, sizeMB)
		// callback: youtube_download|<format_id>
		callbackData := fmt.Sprintf("youtube_download|%s", f.FormatID)
		button := tgbotapi.NewInlineKeyboardButtonData(btnText, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Audio
	if bestAudio != nil {
		sizeMB := bestAudio.Filesize / 1024 / 1024
		btnText := fmt.Sprintf("Audio - %.1fMB", sizeMB)
		callbackData := fmt.Sprintf("youtube_download|%s", bestAudio.FormatID)
		audioButton := tgbotapi.NewInlineKeyboardButtonData(btnText, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(audioButton))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// formatDuration: sekundni HH:MM:SS ko‘rinishiga keltirish
func formatDuration(d float64) string {
	sec := int(d)
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// ------------------------------------------------------
//  2-qadam: Foydalanuvchi tanlagan formatni yuklab, yuborish
// ------------------------------------------------------

// CallbackQuery dan keladigan ma’lumotni (youtube_download|<format_id>) qayta ishlash
func HandleYouTubeDownloadCallback(chatID int64, messageID int, data string, bot *tgbotapi.BotAPI) {
	// data: "youtube_download|<format_id>"
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 {
		log.Println("Noto'g'ri callback data: ", data)
		return
	}
	chosenFormatID := parts[1]

	// 1) Original link + metadata ni keshdan olamiz
	link, ok := YouTubeVideoLinkCache[chatID]
	if !ok {
		log.Printf("ChatID %d uchun link topilmadi", chatID)
		return
	}
	meta, ok := YouTubeVideoInfo[chatID]
	if !ok {
		log.Printf("ChatID %d uchun metadata topilmadi", chatID)
		return
	}

	// 2) Tanlangan formatni lokalga yuklab olamiz
	downloadedFile, err := downloadSpecificFormat(link, chosenFormatID)
	if err != nil {
		log.Printf("Format yuklashda xatolik: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Tanlangan formatni yuklashda xatolik yuz berdi."))
		return
	}

	// 3) 2GB dan oshmaganligini tekshirish
	fileInfo, err := os.Stat(downloadedFile)
	if err == nil {
		if fileInfo.Size() > 50*1024*1024 {
			bot.Send(tgbotapi.NewMessage(chatID, "Kechirasiz, fayl hajmi 50mb dan oshdi. Jo'nata olmayman."))
			// Faylni o'chirishni xohlasangiz:
			os.Remove(downloadedFile)
			return
		}
	}

	// 4) Audio yoki Video ekanligini aniqlash
	isAudio := false
	for _, f := range meta.Formats {
		if f.FormatID == chosenFormatID && f.Vcodec == "none" {
			isAudio = true
			break
		}
	}

	// 5) Yuborish
	if isAudio {
		audioMsg := tgbotapi.NewAudioUpload(chatID, downloadedFile)
		audioMsg.Caption = meta.Title
		if _, err := bot.Send(audioMsg); err != nil {
			log.Printf("Audio yuborishda xatolik: %v", err)
		}
	} else {
		videoMsg := tgbotapi.NewVideoUpload(chatID, downloadedFile)
		videoMsg.Caption = meta.Title
		if _, err := bot.Send(videoMsg); err != nil {
			log.Printf("Video yuborishda xatolik: %v", err)
		}
	}

	RemoveInlineKeyboardAndUpdateCaption(chatID, bot)

	// 7) Faylni o'chirishni istasangiz
	// os.Remove(downloadedFile)
}

// downloadSpecificFormat: `yt-dlp` bilan tanlangan formatni yuklab, lokalga saqlaydi
func downloadSpecificFormat(videoURL, formatID string) (string, error) {
	outName := fmt.Sprintf("youtube_%s.mp4", formatID) // xohlasangiz random nom yarating
	cmd := exec.Command("yt-dlp", "-f", formatID, "-o", outName, videoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("'%s' formatni yuklashda xatolik: %v - %s", formatID, err, string(output))
	}
	return outName, nil
}
