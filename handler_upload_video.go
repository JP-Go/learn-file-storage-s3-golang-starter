package main

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const videoUploadLimit = 1 << 30

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Video not found", nil)
			return
		}
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to you", nil)
		return
	}

	file, header, err := r.FormFile("video")
	defer file.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file.", err)
		return
	}
	contentTypeHeader := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Unsupported video type.", err)
		return
	}

	fileReader := http.MaxBytesReader(w, io.NopCloser(file), videoUploadLimit)
	defer fileReader.Close()
	tempFilename := getAssetPath(mediaType)
	tempFile, err := os.CreateTemp("", tempFilename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file.", err)
		return
	}
	io.Copy(tempFile, file)
	tempFile.Seek(0, io.SeekStart)
	aspectRatio, err := getAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't determine aspect ratio", err)
		return
	}
	newFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to preprocess the video", err)
		return
	}
	tempFile.Close()
	os.Remove(tempFile.Name())
	processedFile, err := os.Open(newFilePath)
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to preprocess the video", err)
		return
	}

	key, err := cfg.uploadToS3Bucket(r.Context(), mediaType, aspectRatio, processedFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload file. Try again later", err)
		return
	}
	videoURL := cfg.getCloudfrontUrl(key)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload file. Try again later", err)
		return
	}
	log.Printf("Uploaded video with id %s to S3 at %s", video.ID, *video.VideoURL)
	respondWithJSON(w, http.StatusAccepted, video)
}
