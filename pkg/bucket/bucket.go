package bucket

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

type Bucket struct {
	Client s3iface.S3API
	Name   string
}

// ReadFile looks through the bucket and reads the first file.
// It returns the contents of the file, its key and/or potentially an error.
func (b *Bucket) ReadFile() (string, string, error) {
	log.WithFields(log.Fields{
		"bucket": b.Name,
	}).Debug("Reading bucket")

	query := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.Name),
	}

	svc := b.Client
	resp, err := b.Client.ListObjectsV2(query)

	if err != nil {
		log.Error("Unable to query bucket")
		return "", "", err
	}

	if len(resp.Contents) < 1 {
		return "", "", errors.New("No files in bucket")
	}

	for _, key := range resp.Contents {

		log.WithFields(log.Fields{
			"file": key.Key,
		}).Debug("Reading File...")

		input := &s3.GetObjectInput{
			Bucket: aws.String(b.Name),
			Key:    key.Key,
		}

		result, err := svc.GetObject(input)

		if err != nil {
			log.Error("Failed to get the file")
			return "", "", err
		}

		log.Info(result)

		body, err := ioutil.ReadAll(result.Body)
		if err != nil {
			log.Error("Unable to read bytes")
			return "", "", err
		}

		return string(body[:]), *key.Key, nil
		break
	}
	return "", "", nil
}

// DeleteObject takes the name of a bucket and a key of of an object in the bucket.
// It will then delete that object if it can find it.
func DeleteObject(sess *session.Session, bucket string, key string) error {
	svc := s3.New(sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": bucket,
			"key":    key,
		}).Error("Failed to delete")
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": bucket,
			"key":    key,
		}).Error("Failed to delete")
		return err
	}

	log.WithFields(log.Fields{
		"bucket": bucket,
		"key":    key,
	}).Info("Successfully deleted")
	return nil
}

func UploadFile(sess *session.Session, bucket string, fileName string, body string) error {
	uploader := s3manager.NewUploader(sess)

	objectPath := "/content/post/" + fileName + ".md"

	fileReader := strings.NewReader(body)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectPath),
		Body:   fileReader,
	})

	if err != nil {
		log.Error("Failed to upload")
		return err
	}

	return nil
}

const tempDir = "/tmp/site/"

func GetObjectsInBucket(sess *session.Session, bucket string) error {
	query := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	os.Mkdir(tempDir, 0777)
	os.Mkdir(tempDir+"public", 0777)

	svc := s3.New(sess)

	truncatedListing := true

	for truncatedListing {
		resp, err := svc.ListObjectsV2(query)

		if err != nil {
			log.WithFields(log.Fields{
				"query": query,
			}).Error("Failed to list objects")
			return err
		}

		err = dowloadObjectsInBucket(resp, sess, bucket)
		if err != nil {
			return err
		}

		truncatedListing = *resp.IsTruncated
	}

	return nil
}

func isDirectory(path string) bool {
	fd, err := os.Stat(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to figure out if path was directory")
		os.Exit(2)
	}
	switch mode := fd.Mode(); {
	case mode.IsDir():
		return true
	case mode.IsRegular():
		return false
	}
	return false
}

// Upload takes all the files in the given path and uploads them to the specified bucket
func Upload(sess *session.Session, bucket string, path string) error {
	path = tempDir + path
	fileList := []string{}
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if isDirectory(path) {
			return nil
		} else {
			fileList = append(fileList, path)
			return nil
		}
	})

	uploader := s3manager.NewUploader(sess)

	for _, file := range fileList {
		actualFile, err := os.Open(file)
		if err != nil {
			log.Error("Unable to open file to write to")
			return err
		}
		defer actualFile.Close()
		file := strings.TrimPrefix(file, path)
		filePath := filepath.ToSlash(file)
		log.Debug(filePath)

		contentType := "text/html"

		if strings.HasSuffix(filePath, ".css") {
			//TODO do this for .less .js and .json files
			contentType = "text/css"
		}

		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(filePath),
			Body:        actualFile,
			ContentType: aws.String(contentType),
		})

		if err != nil {
			log.Error("Unable to upload file")
			return err
		}
	}
	return nil

}

func dowloadObjectsInBucket(bucketObjectsList *s3.ListObjectsV2Output, sess *session.Session, bucket string) error {
	downloader := s3manager.NewDownloader(sess)

	for _, key := range bucketObjectsList.Contents {
		log.Debug(*key.Key)
		destFileName := *key.Key

		if strings.Contains(*key.Key, "/") {
			var dirTree string

			s3FileFullPathList := strings.Split(*key.Key, "/")

			for _, dir := range s3FileFullPathList[:len(s3FileFullPathList)-1] {
				dirTree += "/" + dir
			}
			log.Debug(fmt.Sprintf("making: %s%s", tempDir, dirTree))
			os.MkdirAll(tempDir+dirTree, 0775)
		}

		destFilePath := tempDir + destFileName
		if _, err := os.Stat(destFilePath); !os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"destFilePath": destFilePath,
			}).Debug("Checking if file/dir Exitsts")
		} else {
			log.WithFields(log.Fields{
				"destFilePath": destFilePath,
			}).Debug("Creating file/dir")
			destFile, err := os.Create(destFilePath)
			if err != nil {
				log.WithFields(log.Fields{
					"destFile": destFile,
				}).Error("Failed create file to download into")
				return err
			}

			defer destFile.Close()

			_, err = downloader.Download(destFile, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    key.Key,
			})

			if err != nil {
				log.WithFields(log.Fields{
					"file":     key.Key,
					"destFile": destFile,
				}).Error("Failed to download file")
				return err
			}
		}
	}
	return nil
}
