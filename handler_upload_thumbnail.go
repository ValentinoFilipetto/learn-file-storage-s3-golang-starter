package main

import (
	"fmt"
	"net/http"
	"io"
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
	
	// read the flow of data from `file` and return it as an array of bytes.
	// file is an io.Reader.
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading file", err)
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

	thumbnail := thumbnail{
          data: imageData,
	  mediaType: mediaType,
        }

	videoThumbnails[videoID] = thumbnail

	thumbnailUrl := fmt.Sprintf("http://localhost:%d/api/thumbnails/%s", cfg.port, videoID) 
	
	video.ThumbnailURL = &thumbnailUrl
	
	// we update the video with a new thumbnail
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		delete(videoThumbnails, videoID)
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}


