package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

func isSupportedFileType(filetype string) bool {
	return filetype == "image/png" || filetype == "image/jpeg"
}

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file.", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	defer file.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file.", err)
		return
	}
	contentTypeHeader := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}
	if !isSupportedFileType(mediaType) {
		respondWithError(w, http.StatusBadRequest, "Unsupported media type. Supported types: png, jpeg", err)
		return

	}

	dbVid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Video not found", nil)
			return
		}
	}
	if dbVid.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to you", nil)
		return
	}

	assetPath := getAssetPath(mediaType)
	filename := cfg.getAssetDiskPath(assetPath)
	newFile, err := os.Create(filename)
	defer newFile.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save file", err)
	}

	n, err := io.Copy(newFile, file)
	if err != nil || n == 0 {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save file", err)
	}

	thumbnailURL := cfg.getAssetsURL(assetPath)
	dbVid.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(dbVid)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	respondWithJSON(w, http.StatusOK, dbVid)
}
