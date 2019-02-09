package mail

import (
	"os"
	"path/filepath"
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

type mockedSESAPI struct {
	sesiface.SESAPI
	SendEmailFunc func(*ses.SendEmailInput) (*ses.SendEmailOutput, error)
}

func (m *mockedSESAPI) SendEmail(i *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
	return m.SendEmailFunc(i)
}

func TestSendEmailSendsTheEmailDetailsToTheService(t *testing.T) {
	m := Mail{
		Client: &mockedSESAPI{
			SendEmailFunc: func(i *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
				return &ses.SendEmailOutput{}, nil
			},
		},
	}

	m.SendEmail()
}
