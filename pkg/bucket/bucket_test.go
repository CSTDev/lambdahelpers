package bucket

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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

type mockedListObjects struct {
	s3iface.S3API
	Resp s3.ListObjectsV2Output
}

func (m mockedListObjects) ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return &m.Resp, nil
}

func TestReadFileReturnsErrorWhenBucketIsEmpty(t *testing.T) {
	bucketName := "testBucket"

	resp := s3.ListObjectsV2Output{
		Name: &bucketName,
	}

	b := Bucket{
		Client: mockedListObjects{Resp: resp},
		Name:   bucketName,
	}

	_, _, err := b.ReadFile()
	if err == nil {
		t.Error("Expected error to be returned but didn't receive one")
	}
}

func TestReadFileReturnsTheFirstObjectKey(t *testing.T) {
	bucketName := "testBucket"
	expectedKey := "Object1"
	secondKey := "Object2"

	contents := []s3.Object{
		{
			Key: &expectedKey,
		},
		{
			Key: &secondKey,
		},
	}

	resp := s3.ListObjectsV2Output{
		Name:     &bucketName,
		Contents: &contents,
	}

	b := Bucket{
		Client: mockedListObjects{Resp: resp},
		Name:   bucketName,
	}

	_, key, err := b.ReadFile()
	ok(t, err)

	if key != expectedKey {
		t.Errorf("Expected key not returned. \n Expected: %s \n Received: %s", expectedKey, key)
	}
}
