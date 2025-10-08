package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 10
	r.ParseMultipartForm(maxMemory)

	thumb, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 400, "Failed to get multi-part form file", err)
		return
	}
	defer thumb.Close()

	fileType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil || (fileType != "image/jpeg" && fileType != "image/png") {
		respondWithError(w, 400, "Wrong file type", err)
		return
	}

	extension := strings.ReplaceAll(fileType, "image/", "")
	rng := make([]byte, 32)
	rand.Read(rng)
	fileName := base64.RawURLEncoding.EncodeToString(rng)
	filePath := fmt.Sprintf("/assets/%s.%s", fileName, extension)

	file, err := os.Create("." + filePath)
	if err != nil {
		respondWithError(w, 400, "Failed to create asset file", err)
		return
	}

	_, err = io.Copy(file, thumb)
	if err != nil {
		respondWithError(w, 400, "Failed to read data into asset file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Failed to get video metadata", err)
		return
	}

	tURL := fmt.Sprintf("http://localhost:%s%s", cfg.port, filePath)

	video.ThumbnailURL = &tURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 400, "Failed to get update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
