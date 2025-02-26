package state

var UserStates = make(map[int64]string)

var userMessageIDs = make(map[int64]int)

// Xabar ID'sini saqlash
func SaveMessageID(chatID int64, messageID int) {
	userMessageIDs[chatID] = messageID
}

// Saqlangan xabar ID'sini olish
func GetMessageID(chatID int64) (int, bool) {
	messageID, exists := userMessageIDs[chatID]
	return messageID, exists
}
