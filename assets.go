package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func getAssetPath(mediaType string) string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	ext := getExtensionFromMediaPath(mediaType)
	filename := base64.RawURLEncoding.EncodeToString(randomBytes)
	return fmt.Sprintf("%s%s", filename, ext)
}

func (cfg apiConfig) getAssetsURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg apiConfig) uploadToS3Bucket(ctx context.Context, mimeType, aspectRatio string, file io.Reader) (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	ext := getExtensionFromMediaPath(mimeType)
	filename := fmt.Sprintf("%s/%s%s", aspectRatio, base64.RawURLEncoding.EncodeToString(randomBytes), ext)
	_, err = cfg.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &filename,
		Body:        file,
		ContentType: &mimeType,
	})
	return filename, nil
}
func (cfg apiConfig) getS3Url(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func getExtensionFromMediaPath(mediaType string) string {
	splitted := strings.Split(mediaType, "/")
	if len(splitted) != 2 {
		return ".bin"
	}
	return "." + splitted[1]
}

func (cfg apiConfig) getS3BucketKeyPair(s3ObjectKey string) string {
	return fmt.Sprintf("%s,%s", cfg.s3Bucket, s3ObjectKey)
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(s3Client)
	presignedReq, err := client.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignedReq.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	bucket, key := strings.Split(*video.VideoURL, ",")[0], strings.Split(*video.VideoURL, ",")[1]
	url, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		return video, err
	}
	video.VideoURL = &url
	return video, nil
}
