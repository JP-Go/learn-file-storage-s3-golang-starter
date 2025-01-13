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

	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func (cfg apiConfig) getCloudfrontUrl(key string) string {
	return fmt.Sprintf("%s/%s", cfg.s3CfDistribution, key)
}

func getExtensionFromMediaPath(mediaType string) string {
	splitted := strings.Split(mediaType, "/")
	if len(splitted) != 2 {
		return ".bin"
	}
	return "." + splitted[1]
}
