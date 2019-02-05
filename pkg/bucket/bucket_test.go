package bucket

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

type mockedBucketAPI struct {
	s3iface.S3API
	ListResp         s3.ListObjectsV2Output
	GetResp          s3.GetObjectOutput
	WaitFunc         func(*s3.HeadObjectInput) error
	DeleteObjectFunc func(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	UploadFunc       func(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

func (m mockedBucketAPI) ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return &m.ListResp, nil
}

func (m mockedBucketAPI) GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &m.GetResp, nil
}

func (m mockedBucketAPI) DeleteObject(i *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return m.DeleteObjectFunc(i)
}

func (m mockedBucketAPI) WaitUntilObjectNotExists(i *s3.HeadObjectInput) error {
	return m.WaitFunc(i)
}

func (m mockedBucketAPI) Upload(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return m.UploadFunc(input, options...)
}

func (m mockedBucketAPI) UploadWithContext(aws.Context, *s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return &s3manager.UploadOutput{}, nil
}

// Read single file tests
func TestReadFileReturnsErrorWhenBucketIsEmpty(t *testing.T) {
	bucketName := "testBucket"

	b := Bucket{
		Client: mockedBucketAPI{
			ListResp: s3.ListObjectsV2Output{
				Name: &bucketName,
			},
		},
		Name: bucketName,
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

	b := Bucket{
		Client: mockedBucketAPI{
			ListResp: s3.ListObjectsV2Output{
				Name: &bucketName,
				Contents: []*s3.Object{
					{
						Key: &expectedKey,
					},
					{
						Key: &secondKey,
					},
				},
			},
			GetResp: s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("Hello"))),
			},
		},
		Name: bucketName,
	}

	_, key, err := b.ReadFile()
	ok(t, err)

	if key != expectedKey {
		t.Errorf("Expected key not returned. \n Expected: %s \n Received: %s", expectedKey, key)
	}
}

func TestReadFileReturnsTheBodyOfTheObject(t *testing.T) {
	bucketName := "testBucket"
	key := "Object1"
	expectedBody := "Hello"

	b := Bucket{
		Client: mockedBucketAPI{
			ListResp: s3.ListObjectsV2Output{
				Name: &bucketName,
				Contents: []*s3.Object{
					{
						Key: &key,
					},
				},
			},
			GetResp: s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("Hello"))),
			},
		},
		Name: bucketName,
	}

	body, _, err := b.ReadFile()
	ok(t, err)

	if body != expectedBody {
		t.Errorf("Expected key not returned. \n Expected: %s \n Received: %s", expectedBody, body)
	}
}

// Delete tests
func TestDeleteObjectCallsDeleteAndWaitsForObjectToNotExist(t *testing.T) {
	isDeleteCalled := false
	isWaitCalled := false

	b := Bucket{
		Client: mockedBucketAPI{
			DeleteObjectFunc: func(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				isDeleteCalled = true
				return &s3.DeleteObjectOutput{}, nil
			},
			WaitFunc: func(*s3.HeadObjectInput) error {
				isWaitCalled = true
				return nil
			},
		},
		Name: "TestBucket",
	}

	err := b.DeleteObject("Object1")

	ok(t, err)

	if !isDeleteCalled {
		t.Error("Expected DeleteObject to be called, it wasn't")
	}

	if !isWaitCalled {
		t.Error("Expected Wait to be called, it wasn't")
	}
}

// Upload tests
func TestUploadCallsUploaderWithBucketAndKey(t *testing.T) {
	isUploadCalled := false
	var bucketCalled string
	var keyCalled string
	expectedBucket := "TestBucket"
	expectedKey := "/content/post/TestFile.md"

	b := Bucket{
		Uploader: mockedBucketAPI{
			UploadFunc: func(i *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
				isUploadCalled = true
				bucketCalled = *i.Bucket
				keyCalled = *i.Key
				return &s3manager.UploadOutput{}, nil
			},
		},
		Name: expectedBucket,
	}

	err := b.UploadFile("TestFile", "Some conent")

	ok(t, err)
	if !isUploadCalled {
		t.Error("Expected Upload to be called, it wasn't")
		t.FailNow()
	}

	if bucketCalled != expectedBucket {
		t.Errorf("Expected Bucket: %s \n Actual Bucket: %s", expectedBucket, bucketCalled)
		t.FailNow()
	}

	if keyCalled != expectedKey {
		t.Errorf("Expected Key: %s \n Actual Key: %s", expectedKey, keyCalled)
		t.FailNow()
	}
}
