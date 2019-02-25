package mail

import (
	"strings"

	"github.com/DusanKasan/parsemail"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"

	log "github.com/sirupsen/logrus"
)

const subject = "S3Reader Raw"
const charSet = "UTF-8"

type SESMail struct {
	Client sesiface.SESAPI
}

func (m *SESMail) SendMail(recipient string, sender string, body string) error {

	log.WithFields(log.Fields{
		"sender":    sender,
		"recipient": recipient,
	}).Debug("Sending email")

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
	}

	_, err := m.Client.SendEmail(input)

	if err != nil {
		log.Error("Failed to send email")
		return err
	}

	return nil
}

type Message struct {
	Subject string
	Body    string
}

func ParseBody(email string) Message {
	log.Debug("Parsing body")

	p := strings.NewReader(email)
	emailOut, err := parsemail.Parse(p)
	if err != nil {
		panic(err)
	}

	message := &Message{
		Subject: emailOut.Subject,
		Body:    emailOut.TextBody,
	}

	return *message
}
