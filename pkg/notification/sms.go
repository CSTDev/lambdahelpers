package notification

import (
	log "github.com/sirupsen/logrus"
)

func SendMessage(message string, number string) bool {
	log.Info("Sending message")
	return false
}
