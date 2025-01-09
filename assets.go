package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func getExtensionFromMediaPath(mediaType string) string {
	splitted := strings.Split(mediaType, "/")
	if len(splitted) != 2 {
		return ".bin"
	}
	return "." + splitted[1]
}
