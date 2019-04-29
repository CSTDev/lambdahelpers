package notification

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	log "github.com/sirupsen/logrus"
)

// SMS contains the AWS SNS client used to send the messages with
type SMS struct {
	Client snsiface.SNSAPI
}

// SendMessage sends the provided message to the provided number
func (s *SMS) SendMessage(message string, number string) error {
	log.Info("Sending message")
	if message == "" || number == "" {
		log.WithFields(log.Fields{
			"message": message,
			"number":  number,
		}).Error("Missing message or phone number")
		return errors.New("Missing message or phone number")
	}
	messageParams := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(number),
	}
	resp, err := s.Client.Publish(messageParams)
	if err != nil {
		log.Error("Failed to send text message")
		return err
	}

	log.Debug(resp)

	return nil
}
