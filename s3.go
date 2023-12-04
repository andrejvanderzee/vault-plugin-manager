package main

import (
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

func newAWSSession(region string) (*session.Session, error) {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *PluginSyncSession) s3ListPlugins() ([]Plugin, error) {

	svc := s3.New(s.AWSSession)
	res, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: &s.S3Bucket,
	})
	if err != nil {
		return nil, err
	}

	plugs := make([]Plugin, 0, len(res.Contents))
	for _, item := range res.Contents {
		data, err := svc.HeadObject(&s3.HeadObjectInput{
			Bucket: &s.S3Bucket,
			Key:    item.Key,
		})

		if err != nil {
			return nil, errors.Wrapf(err, "failed to get head object for plugin %s", *item.Key)
		}

		plugType, ok := data.Metadata["Type"]
		if !ok {
			return nil, errors.Errorf("expected 'Type' in metadata for plugin: %s", *item.Key)
		}

		plugSum := data.Metadata["Sha256sum"]
		if !ok {
			return nil, errors.Errorf("expected 'Sha256sum' in metadata for plugin: %s", *item.Key)
		}

		plug := Plugin{Name: *item.Key,
			Type:   *plugType,
			SHA256: *plugSum,
		}
		plugs = append(plugs, plug)
	}

	return plugs, nil
}

func (s *PluginSyncSession) s3DownloadPlugin(pluginName string) error {

	path := filepath.Join(s.PluginPath, pluginName)
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", path)
	}

	defer file.Close()
	downloader := s3manager.NewDownloader(s.AWSSession)

	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(s.S3Bucket),
			Key:    aws.String(pluginName),
		})
	if err != nil {
		return errors.Wrapf(err, "failed to download s3://%s/%s", s.S3Bucket, pluginName)
	}

	return nil
}
