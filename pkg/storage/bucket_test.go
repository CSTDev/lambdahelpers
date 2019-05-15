package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cstdev/lambdahelpers/pkg/s3/manager"
	log "github.com/sirupsen/logrus"
)

const basePath = "testdata"
const destFilePath = basePath + "/output"
const srcFilePath = basePath + "/input"

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})
	retCode := m.Run()
	clearDirectories()
	os.Exit(retCode)
}

func clearDirectories() {
	os.RemoveAll(destFilePath)
	os.RemoveAll(srcFilePath + "/benchmarkUpload/")
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
	manager.S3Manager
	ListObjectsFunc  func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	GetObjectFunc    func(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
	WaitFunc         func(*s3.HeadObjectInput) error
	DeleteObjectFunc func(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	UploadFunc       func(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	DownloadFunc     func(io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error)
}

func (m mockedBucketAPI) ListObjectsV2(i *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return m.ListObjectsFunc(i)
}

func (m mockedBucketAPI) GetObject(i *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(i)
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

func (m mockedBucketAPI) Download(w io.WriterAt, i *s3.GetObjectInput, options ...func(*s3manager.Downloader)) (int64, error) {
	return m.DownloadFunc(w, i, options...)
}

// Read single file tests
func TestReadFileReturnsErrorWhenBucketIsEmpty(t *testing.T) {
	bucketName := "testBucket"

	b := Bucket{
		Client: mockedBucketAPI{
			ListObjectsFunc: func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Name: &bucketName,
				}, nil
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
			ListObjectsFunc: func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Name: &bucketName,
					Contents: []*s3.Object{
						{
							Key: &expectedKey,
						},
						{
							Key: &secondKey,
						},
					},
				}, nil
			},
			GetObjectFunc: func(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: ioutil.NopCloser(bytes.NewReader([]byte("Hello"))),
				}, nil
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
			ListObjectsFunc: func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Name: &bucketName,
					Contents: []*s3.Object{
						{
							Key: &key,
						},
					},
				}, nil
			},
			GetObjectFunc: func(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: ioutil.NopCloser(bytes.NewReader([]byte("Hello"))),
				}, nil
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
func TestUploadFileCallsUploaderWithBucketAndKey(t *testing.T) {
	isUploadCalled := false
	var bucketCalled string
	var keyCalled string
	expectedBucket := "TestBucket"
	expectedKey := "/content/post/TestFile.md"

	b := Bucket{
		Manager: mockedBucketAPI{
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

func TestUploadSendsAllFilesInDirectoryToUpload(t *testing.T) {
	var keys []string
	keyFilePath := "testdata/input/testUpload"
	expectedKeys := []string{keyFilePath + "/Object1.txt", keyFilePath + "/Object2.md"}
	bucket := Bucket{
		Name: "DestBucket",
		Manager: mockedBucketAPI{
			UploadFunc: func(i *s3manager.UploadInput, up ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
				log.WithFields(log.Fields{
					"key": *i.Key,
				}).Debug("Uploading key")
				keys = append(keys, *i.Key)
				buf := new(bytes.Buffer)
				buf.ReadFrom(i.Body)
				return &s3manager.UploadOutput{}, nil
			},
		},
	}

	err := bucket.Upload(srcFilePath)
	ok(t, err)

	var found bool
	for _, key := range keys {
		for _, expectedKey := range expectedKeys {
			log.WithFields(log.Fields{
				"Expected": expectedKey,
				"Uploaded": key,
			}).Debug("Checking object keys match")
			if expectedKey == key {
				found = true
				break
			}
			found = false
		}
		if !found {
			t.Errorf("Expected key to be found in uploaded keys: %s ", key)
		}
	}
}

func generateFilesToUpload(number int, benchmarkDir string) {
	clearDirectories()
	os.Mkdir(benchmarkDir, 0777)
	for n := 0; n < number; n++ {
		if n%2 == 0 {
			subDir, err := ioutil.TempDir(benchmarkDir, "subdir")
			if err != nil {
				panic(err)
			}
			file, err := ioutil.TempFile(subDir, "testfile*.md")
			if err != nil {
				panic(err)
			}
			buff := make([]byte, 1000)
			ioutil.WriteFile(file.Name(), buff, 0666)
		} else {
			file, err := ioutil.TempFile(benchmarkDir, "testfile*.md")
			if err != nil {
				panic(err)
			}
			buff := make([]byte, 1000)
			ioutil.WriteFile(file.Name(), buff, 0666)
		}

	}
}
func BenchmarkUploadSendsAllFilesInDirectoryToUpload(b *testing.B) {
	var keys []string
	srcFilePath := srcFilePath + "/benchmarkUpload"
	generateFilesToUpload(100, srcFilePath)
	b.ResetTimer()
	bucket := Bucket{
		Name: "DestBucket",
		Manager: mockedBucketAPI{
			UploadFunc: func(i *s3manager.UploadInput, up ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
				keys = append(keys, *i.Key)
				return &s3manager.UploadOutput{}, nil
			},
		},
	}

	for n := 0; n < b.N; n++ {
		err := bucket.Upload(srcFilePath)
		ok(b, err)
	}

}

// Download Objects tests

func downloadObjects(objectKey string, expectedPath string, tb testing.TB) {
	bucketName := "TestBucket"
	isTruncated := false
	bucket := Bucket{
		Client: mockedBucketAPI{
			ListObjectsFunc: func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Name: &bucketName,
					Contents: []*s3.Object{
						{
							Key: &objectKey,
						},
					},
					IsTruncated: &isTruncated,
				}, nil
			},
		},
		Manager: mockedBucketAPI{
			DownloadFunc: func(io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error) {
				return 0, nil
			},
		},
		Name: bucketName,
	}

	err := bucket.DownloadAllObjectsInBucket(destFilePath)
	ok(tb, err)

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		tb.Error("Expected destination object to be created.")
	}
}
func TestDownloadObjectsMakesTheDirectoryToDownloadTo(t *testing.T) {

	objectKey := "Object1"
	expectedPath := destFilePath
	downloadObjects(objectKey, expectedPath, t)

}

func TestDownloadCreatesObjectInSubDirectory(t *testing.T) {
	objectKey := "/posts/Object1"
	expectedPath := destFilePath + objectKey
	downloadObjects(objectKey, expectedPath, t)

}

func TestDownloadWillCreateOtherDirsWhenTheyArePassed(t *testing.T) {
	bucketName := "TestBucket"
	objectKey := "Object1"
	isTruncated := false
	bucket := Bucket{
		Client: mockedBucketAPI{
			ListObjectsFunc: func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Name: &bucketName,
					Contents: []*s3.Object{
						{
							Key: &objectKey,
						},
					},
					IsTruncated: &isTruncated,
				}, nil
			},
		},
		Manager: mockedBucketAPI{
			DownloadFunc: func(io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error) {
				return 0, nil
			},
		},
		Name: bucketName,
	}

	err := bucket.DownloadAllObjectsInBucket(destFilePath, "public", "not-public")
	ok(t, err)

	expectedPath := destFilePath + "/public"

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected %s to be created.", expectedPath)
	}

	expectedPath = destFilePath + "/not-public"

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected %s to be created.", expectedPath)
	}
}
