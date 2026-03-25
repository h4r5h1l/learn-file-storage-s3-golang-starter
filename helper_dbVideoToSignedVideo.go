package main

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignParams := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	presignedRequest, err := presignClient.PresignGetObject(context.Background(), presignParams, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignedRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(databaseVideo database.Video) (database.Video, error) {
	if databaseVideo.VideoURL == nil {
		return databaseVideo, nil
	}
	bucket, key := strings.Split(*databaseVideo.VideoURL, ",")[0], strings.Split(*databaseVideo.VideoURL, ",")[1]
	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 15*time.Minute)
	if err != nil {
		return databaseVideo, err
	}
	databaseVideo.VideoURL = &signedURL
	return databaseVideo, nil
}
