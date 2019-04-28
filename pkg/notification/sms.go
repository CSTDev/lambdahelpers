package notification

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	log "github.com/sirupsen/logrus"
)

type SMS struct {
	Client snsiface.SNSAPI
}

func (s *SMS) SendMessage(message string, number string) error {
	log.Info("Sending message")
	messageParams := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(number),
	}
	resp, err := s.Client.Publish(messageParams)
	if err != nil {
		log.Error("Failed to send text message")
		return err
	}

	log.Info(resp)

	return nil
}
