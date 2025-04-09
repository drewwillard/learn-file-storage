package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusConflict, "couldn't open form file", err)
		return
	}

	contentType := header.Header.Get("Content-Type")
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusConflict, "couldn't read from body", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "user isn't video owner", err)
		return
	}

	cleanType, _, err := mime.ParseMediaType(contentType)
	if err != nil || (cleanType != "image/jpeg" && cleanType != "image/png") {
		respondWithError(w, http.StatusBadRequest, "unsupported image type", nil)
		return
	}
	ext := "." + strings.SplitAfter(cleanType, "/")[1]

	newFileBytes := make([]byte, 32)
	_, err = rand.Read(newFileBytes)
	randString := base64.RawURLEncoding.EncodeToString(newFileBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't generate random bytes", err)
		return
	}

	filename := randString + ext
	filepath := filepath.Join(cfg.assetsRoot, filename)

	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't write file", err)
		return
	}
	newUrl := "http://localhost:8091/assets/" + filename
	video.ThumbnailURL = &newUrl

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusConflict, "couldn't update video", err)
		return
	}
	fmt.Println("Success with url:", video.ThumbnailURL)
	respondWithJSON(w, http.StatusOK, video)
}
