package main

import (
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
	// Get the user ID from the JWT in the Authorization header
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
	// Get the video ID from the URL path and fetch the video data from the database
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video data", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "User doesn't have access to this video", nil)
		return
	}

	const maxMemory = 10 << 20 // 10 MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	src, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail from form data", err)
		return
	}
	defer src.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't determine media type of thumbnail", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" && mediaType != "image/gif" {
		respondWithError(w, http.StatusBadRequest, "Not an allowed image type", nil)
		return
	}
	filetype := strings.Split(mediaType, "/")[1]
	filename := fmt.Sprintf("/%s.%s", videoID, filetype)
	filepath := filepath.Join(cfg.assetsRoot, filename)
	dest, err := os.Create(filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save thumbnail file", err)
		return
	}

	dataURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &dataURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video with thumbnail URL", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
