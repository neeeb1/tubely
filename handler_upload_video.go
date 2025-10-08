package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxSize = 1 << 30
	http.MaxBytesReader(w, r.Body, maxSize)

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
		respondWithError(w, http.StatusUnauthorized, "Failed to get video metadata", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Failed to get video metadata", err)
		return
	}

	r.ParseMultipartForm(maxSize)
	videoFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 400, "Failed to get multi-part form file", err)
		return
	}
	defer videoFile.Close()

	fileType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil || fileType != "video/mp4" {
		respondWithError(w, 400, "Wrong file type", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 400, "Failed to get create temp file", err)
		return
	}
	defer os.Remove("tubely-upload.mp4")
	defer tempFile.Close()

	io.Copy(tempFile, videoFile)
	tempFile.Seek(0, io.SeekStart)

	rng := make([]byte, 32)
	rand.Read(rng)
	key := base64.RawURLEncoding.EncodeToString(rng) + ".mp4"
	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Body:        tempFile,
		Key:         &key,
		ContentType: &fileType,
	}

	cfg.s3Client.PutObject(r.Context(), &params)

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	video.VideoURL = &url

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 400, "Failed to get update video meta data", err)
		return
	}
}
