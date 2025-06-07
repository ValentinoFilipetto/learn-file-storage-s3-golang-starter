package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// we want to restrict the size of the incoming request body.
	const uploadLimit = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)

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

	fmt.Println("uploading video", videoID, "by user", userID)

	// get video by ID
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// we specifically try and get the video as a multipart file
	multipartVideo, header, err := r.FormFile("video")
	if err != nil {
	    respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
	    return
	}

	// file is a stream, hence eventually it will need to be closed.
	defer multipartVideo.Close()

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for video", nil)
		return
	}

	// get media type from content type header
	mediaType, _, err = mime.ParseMediaType(mediaType)
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusInternalServerError, "Wrong media type received", nil)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while getting the media type", nil)
		return
	}

	// create temporary file in the location indicated by the filepath
	// "" as directory uses the system default, e.g. /tmp/
	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	fmt.Println(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create temporary file", err)
		return
	}
	// file is a stream, hence eventually it will need to be closed and removed.
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// copy file content into new empty file
	_, err = io.Copy(tmpFile, multipartVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy content to file", err)
		return
	}

	// this will allow us to read the file again from the beginning
	tmpFile.Seek(0, io.SeekStart)

	aspectRatio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not get the aspect ratio of video", err)
		return
	}

	var prefix string
    if aspectRatio == "16:9" {
		prefix = "landscape"
	} else if aspectRatio == "9:16" {
		prefix = "portrait"
	} else {
		prefix = "other"
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while generating random bytes", nil)
		return
	}
	encodedKey := hex.EncodeToString(key)
	myKey := fmt.Sprintf("%s%s.%s", prefix, encodedKey, strings.Split(mediaType, "/")[1])

	// we put the video in the S3 bucket
	params := s3.PutObjectInput{
		Bucket:  &cfg.s3Bucket,
		Key: &myKey,
		Body: tmpFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		fmt.Printf("S3 PutObject error: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, "Error while putting object to S3", nil)
		return
	}
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, myKey)
	video.VideoURL = &videoURL

	// we update the video with a new videoURL, i.e. the location of the video in the S3 bucket
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
