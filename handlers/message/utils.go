package message

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// parseIDs is a helper function to convert Discord string IDs to int64.
func parseIDs(messageID, userID, guildID, channelID string) (int64, int64, int64, int64, error) {
	msgID, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing message ID %s: %w", messageID, err)
	}
	var uID int64
	if userID != "" {
		uID, err = strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("error parsing user ID %s: %w", userID, err)
		}
	}
	gID, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing guild ID %s: %w", guildID, err)
	}
	cID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing channel ID %s: %w", channelID, err)
	}
	return msgID, uID, gID, cID, nil
}

// getAttachmentsJSON is a helper function to convert attachment objects to a JSON string.
func getAttachmentsJSON(attachments []*discordgo.MessageAttachment) string {
	if len(attachments) == 0 {
		return ""
	}
	var urls []string
	for _, attachment := range attachments {
		urls = append(urls, attachment.URL)
	}
	jsonData, err := json.Marshal(urls)
	if err != nil {
		log.Printf("Error marshalling attachments to JSON: %v", err)
		return ""
	}
	return string(jsonData)
}
