package s3managerinterface

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Manager interface groups the provided UploaderAPI and DownloaderAPI interfaces
type S3Manager interface {
	Download(io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error)
	DownloadWithContext(aws.Context, io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error)

	Upload(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	UploadWithContext(aws.Context, *s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}
