package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	http.MaxBytesReader(w, r.Body, maxMemory)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing video ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error getting token", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get userID from jwt", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 404, "could not find video", err)
		return
	}

	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	file, _, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "issue parsing uploaded video", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType("video/mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "file wrong media type", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error creating temp file", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error copying file", err)
		return
	}

	aspectRatio := ""
	ratio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get video aspect ratio", err)
		return
	}
	if ratio == "16:9" {
		aspectRatio = "landscape"
	} else if ratio == "9:16" {
		aspectRatio = "portrait"
	} else {
		aspectRatio = "other"
	}
	/*

		_, err = tmpFile.Seek(0, io.SeekStart)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "error resetting pointer to start of temp file", err)
			return
		}
	*/

	//preprocessing video
	newPath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not process video fast", err)
		fmt.Printf("%v\n", err)
		return
	}

	processedFile, err := os.Open(newPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not open processed file", err)
		return
	}
	defer processedFile.Close()

	key := make([]byte, 32)
	rand.Read(key)

	url := fmt.Sprintf("%v/%v.mp4", aspectRatio, base64.RawURLEncoding.EncodeToString(key))

	s3BucketParams := s3.PutObjectInput{
		Bucket:      aws.String("tubely-08211280"),
		Key:         &url,
		Body:        processedFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &s3BucketParams)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error putting video in s3 bucket", err)
		return
	}
	s3URL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, url)
	video.VideoURL = &s3URL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error updating video", err)
		return
	}
	respondWithJSON(w, 200, nil)

}
