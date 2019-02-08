package manager

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

type BucketManager struct {
	S3Manager
	Uploader   s3manager.Uploader
	Downloader s3manager.Downloader
}

func (b *BucketManager) Download(w io.WriterAt, i *s3.GetObjectInput, opts ...func(*s3manager.Downloader)) (int64, error) {
	return b.Downloader.Download(w, i, opts...)
}

func (b *BucketManager) Upload(i *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return b.Uploader.Upload(i, opts...)
}
