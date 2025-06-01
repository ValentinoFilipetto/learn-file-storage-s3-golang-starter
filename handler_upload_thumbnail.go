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

	// we received a request as multipart/form-data, which contains a thumbnail.
	// Here we separate the different part of it.
	// maxMemory is needed to have a constraint of the size of what we are processing.
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// we specifically try and get the thumbnail
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
	       respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
	       return
	}
	// file is a stream, hence eventually it will need to be closed.
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	// get media type from content type header
	mediaType, _, err = mime.ParseMediaType(mediaType)
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusInternalServerError, "Wrong media type received", nil)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while getting the media type", nil)
		return
	}

	// get video by id
	video, err  := cfg.db.GetVideo(videoID)
	if err != nil {
       		respondWithError(w, http.StatusBadRequest, "Could not find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

	// create filepath to store imageData in the filesystem
	filepath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoID.String(), strings.Split(mediaType, "/")[1]))

	// create empty new file in the location indicated by the filepath
	emptyFile, err := os.Create(filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create file", err)
		return
	}
	// file is a stream, hence eventually it will need to be closed.
	defer emptyFile.Close()

	// copy file content into new empty file
	_, err = io.Copy(emptyFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy content to file", err)
		return
	}

	thumbnail_url := fmt.Sprintf("http://localhost:%s/%s", cfg.port, filepath)

	video.ThumbnailURL = &thumbnail_url

	// we update the video with a new thumbnail URL, mot the image itself for now
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}


