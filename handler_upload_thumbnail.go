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
	// Authenticate the request via JWT bearer token
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
	// Parse and validate the video ID from the URL path
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	// Fetch the video record and confirm the requesting user owns it
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video data", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "User doesn't have access to this video", nil)
		return
	}

	// Parse the multipart form data, capping memory usage at 10 MB
	const maxMemory = 10 << 20 // 10 MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	// Extract the thumbnail file from the form
	src, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail from form data", err)
		return
	}
	defer src.Close()

	// Validate that the upload is an allowed image type
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't determine media type of thumbnail", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" && mediaType != "image/gif" {
		respondWithError(w, http.StatusBadRequest, "Not an allowed image type", nil)
		return
	}

	// Generate a random filename and build the full disk path
	filetype := strings.Split(mediaType, "/")[1]
	var randomPath [32]byte
	_, err = rand.Read(randomPath[:])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate thumbnail name", err)
		return
	}
	pathString := base64.RawURLEncoding.EncodeToString(randomPath[:])
	filename := fmt.Sprintf("/%s.%s", pathString, filetype)
	filepath := filepath.Join(cfg.assetsRoot, filename)

	// Create the destination file on disk and stream the upload into it
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

	// Persist the public URL for the thumbnail on the video record
	dataURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &dataURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video with thumbnail URL", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
