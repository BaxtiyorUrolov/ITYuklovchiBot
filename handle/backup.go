package handle

import (
	"bytes"
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"os/exec"
	"time"
	"yuklovchiBot/storage"
)

func HandleBackup(db *sql.DB, botInstance *tgbotapi.BotAPI) {
	// Hozirgi sana
	currentTime := time.Now().Format("2006-01-02")
	backupDir := "./backups"
	backupFile := fmt.Sprintf("%s/backup_%s.sql", backupDir, currentTime)

	// Backup katalogini yaratish
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
			log.Printf("Backup katalogini yaratib bo'lmadi: %v", err)
			return
		}
	}

	// PostgreSQL backupni yaratish
	cmd := exec.Command("pg_dump", "-U", "godb", "-d", "grscanbot", "-f", backupFile)
	cmd.Env = append(os.Environ(), "PGPASSWORD=0208") // Parolni muhit o'zgaruvchisi sifatida o'rnatish

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Backup yaratishda xatolik: %v, %s", err, stderr.String())
		return
	}

	log.Printf("Backup muvaffaqiyatli yaratildi: %s", backupFile)

	// Adminlarning IDlarini olish
	adminIDs, err := storage.GetAdmins(db)
	if err != nil {
		log.Printf("Adminlarni olishda xatolik: %v", err)
		return
	}

	for _, chatID := range adminIDs {
		SendBackupToAdmin(chatID, backupFile, botInstance)
	}
}

// SendBackupToAdmin sends a backup file to a specific admin
func SendBackupToAdmin(chatID int64, filePath string, botInstance *tgbotapi.BotAPI) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Backup faylni ochib bo'lmadi: %v", err)
		return
	}
	defer file.Close()

	msg := tgbotapi.NewDocumentUpload(chatID, tgbotapi.FileReader{
		Name:   filePath,
		Reader: file,
		Size:   -1,
	})

	if _, err := botInstance.Send(msg); err != nil {
		log.Printf("Admin (%d) uchun backupni yuborishda xatolik: %v", chatID, err)
	} else {
		log.Printf("Admin (%d) uchun backup muvaffaqiyatli yuborildi.", chatID)
	}
}
