package storage

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cstdev/lambdahelpers/pkg/s3/manager"
	"github.com/karrick/godirwalk"
	log "github.com/sirupsen/logrus"
)

type Bucket struct {
	Client  s3iface.S3API
	Manager manager.S3Manager
	Name    string
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

		result, err := b.Client.GetObject(input)

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
func (b *Bucket) DeleteObject(key string) error {
	_, err := b.Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(b.Name),
		Key:    aws.String(key),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": b.Name,
			"key":    key,
		}).Error("Failed to delete")
		return err
	}

	err = b.Client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(b.Name),
		Key:    aws.String(key),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": b.Name,
			"key":    key,
		}).Error("Failed to delete")
		return err
	}

	log.WithFields(log.Fields{
		"bucket": b.Name,
		"key":    key,
	}).Info("Successfully deleted")
	return nil
}

// UploadFile will write a string to an object in a bucket.
// Currently writes the object with the prefix /content/post and
// suffix .md
// It takes the body, and a fileName as the key
func (b *Bucket) UploadFile(fileName string, body string) error {
	objectPath := "/content/post/" + fileName + ".md"

	fileReader := strings.NewReader(body)

	_, err := b.Manager.Upload(&s3manager.UploadInput{
		Bucket: aws.String(b.Name),
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

// DownloadAllObjectsInBucket downloads all objects it finds in a bucket
// to /tmp/site
func (b *Bucket) DownloadAllObjectsInBucket(destDir string, otherDirs ...string) error {
	query := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.Name),
	}

	if string(destDir[len(destDir)-1:]) != "/" {
		destDir += "/"
	}

	os.Mkdir(destDir, 0777)

	if len(otherDirs) > 0 {
		for _, dir := range otherDirs {
			os.Mkdir(destDir+dir, 0777)
		}
	}

	truncatedListing := true

	for truncatedListing {
		resp, err := b.Client.ListObjectsV2(query)

		if err != nil {
			log.WithFields(log.Fields{
				"query": query,
			}).Error("Failed to list objects")
			return err
		}

		err = dowloadObjectsInBucket(resp, *b, destDir)
		if err != nil {
			return err
		}

		truncatedListing = *resp.IsTruncated
	}

	return nil
}

func uploadFile(inFile string, path string, b Bucket) error {
	actualFile, err := os.Open(inFile)
	if err != nil {
		log.Error("Unable to open file to write to")
		return err
	}
	defer actualFile.Close()
	file := strings.TrimPrefix(filepath.ToSlash(inFile), filepath.ToSlash(path))
	filePath := filepath.ToSlash(file)
	log.WithFields(log.Fields{
		"file":     file,
		"filePath": filePath,
		"inFile":   inFile,
	}).Debug("File being uploaded")

	contentType := "text/html"

	if strings.HasSuffix(filePath, ".css") {
		// TODO do this for .less .js and .json files
		contentType = "text/css"
	}

	_, err = b.Manager.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(b.Name),
		Key:         aws.String(filePath),
		Body:        actualFile,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		log.Error("Unable to upload file")
		return err
	}
	return nil
}

// Upload takes all the files in the given path and uploads them to the specified bucket
func (b *Bucket) Upload(path string) error {
	err := godirwalk.Walk(path, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !isDirectory(osPathname) {
				log.WithFields(log.Fields{
					"osPathName": osPathname,
					"path":       path,
				}).Debug()
				return uploadFile(osPathname, path, *b)
			}
			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to get file paths to upload")
		return err
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

func dowloadObjectsInBucket(bucketObjectsList *s3.ListObjectsV2Output, b Bucket, destDir string) error {

	for _, key := range bucketObjectsList.Contents {
		log.Debug(*key.Key)
		destFileName := *key.Key

		if strings.Contains(*key.Key, "/") {
			var dirTree string

			s3FileFullPathList := strings.Split(*key.Key, "/")

			for _, dir := range s3FileFullPathList[:len(s3FileFullPathList)-1] {
				dirTree += "/" + dir
			}
			log.Debug(fmt.Sprintf("making: %s%s", destDir, dirTree))
			os.MkdirAll(destDir+dirTree, 0775)
		}

		destFilePath := destDir + destFileName
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

			_, err = b.Manager.Download(destFile, &s3.GetObjectInput{
				Bucket: aws.String(b.Name),
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
