package notification

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
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

type mockSMSAPI struct {
	snsiface.SNSAPI
	PublishFunc func(*sns.PublishInput) (*sns.PublishOutput, error)
}

func (s *mockSMSAPI) Publish(i *sns.PublishInput) (*sns.PublishOutput, error) {
	return s.PublishFunc(i)
}

func TestPublishIsCalledWithMessageTextAndNumber(t *testing.T) {
	var message string
	var number string
	expectedMessage := "Hello Test"
	expectedNumber := "+12345678910"
	s := SMS{
		Client: &mockSMSAPI{
			PublishFunc: func(i *sns.PublishInput) (*sns.PublishOutput, error) {
				message = *i.Message
				number = *i.PhoneNumber
				return &sns.PublishOutput{}, nil
			},
		},
	}

	err := s.SendMessage(expectedMessage, expectedNumber)
	ok(t, err)

	equals(t, expectedMessage, message)
	equals(t, expectedNumber, number)
}
