package mail

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})
	retCode := m.Run()
	os.Exit(retCode)
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.WithFields(log.Fields{
			"file":  filepath.Base(file),
			"line":  line,
			"error": err.Error(),
		}).Error("unexpected error")
		tb.FailNow()
	}
}

func equals(tb testing.TB, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		tb.Errorf("Expected: %s \n Actual: %s", expected, actual)
	}
}

type mockedSESAPI struct {
	sesiface.SESAPI
	SendEmailFunc func(*ses.SendEmailInput) (*ses.SendEmailOutput, error)
}

func (m *mockedSESAPI) SendEmail(i *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
	return m.SendEmailFunc(i)
}

func TestSendEmailSendsTheEmailDetailsToTheService(t *testing.T) {
	var to string
	var from string
	var body string
	expectedTo := "recipient@test.com"
	expectedFrom := "sender@test.com"
	expectedBody := "body"
	m := Mail{
		Client: &mockedSESAPI{
			SendEmailFunc: func(i *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
				to = *i.Destination.ToAddresses[0]
				from = *i.Source
				body = *i.Message.Body.Text.Data

				return &ses.SendEmailOutput{}, nil
			},
		},
	}

	err := m.SendMail(expectedTo, expectedFrom, expectedBody)
	ok(t, err)

	equals(t, expectedTo, to)
	equals(t, expectedFrom, from)
	equals(t, expectedBody, body)
}
