package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type FFStream struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type FFProbeResult struct {
	Streams []FFStream `json:"streams"`
}

const landscape = "landscape"
const portrait = "portrait"
const other = "other"

func getAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var res FFProbeResult
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		return "", err
	}
	var stream FFStream
	for i := 0; i < len(res.Streams); i++ {
		if res.Streams[i].Width != 0 && res.Streams[i].Height != 0 {
			stream = res.Streams[i]
			break
		}
	}
	return calculateAspectRatio(stream.Width, stream.Height), nil
}

func calculateAspectRatio(width, height int) string {
	switch {
	case int(float64(width)/16*9) == height:
		return landscape
	case int(float64(width)/9*16) == height:
		return portrait
	default:
		return other
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	newFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return newFilePath, nil
}
