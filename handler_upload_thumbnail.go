package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	//get the video id and convert to a uuid type
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	//get the token from the header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	//get userID from the token
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	//parses the request body up to 10MB(maxMemory)
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not Parse mutlipart form", err)
		return
	}

	//getting first file and file header (multipart.file and *multipart.fileheader)
	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not file or file header", err)
		return
	}
	defer file.Close()

	//get the media type
	mediaType := fileHeader.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "failed to get media type", err)
		return
	}

	//get the information from the video using the videoID provided
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 404, "could not find video", err)
		return
	}

	//check the userID is the ID that made the video or has access to make changes
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	//make the picture into a string and store it in the video thumbnail url
	//pictureData := base64.StdEncoding.EncodeToString(myThumbnail.data)
	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(extensions) < 1 {
		respondWithError(w, http.StatusBadRequest, "could not get extensions", err)
		return
	}

	parseMediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error with parsing media type", err)
		return
	}

	if (parseMediaType != "image/jpeg") && (parseMediaType != "image/png") {
		respondWithError(w, http.StatusBadRequest, "not of type image/jpeg or image/png", nil)
		return
	}

	videoWExtension := fmt.Sprintf("%v%v", videoID.String(), extensions[0])
	baseURL := fmt.Sprintf("http://localhost:%v", cfg.port)
	localPath := filepath.Join(cfg.assetsRoot, videoWExtension)

	fullPath := fmt.Sprintf("%v/%v", baseURL, localPath)
	//fullPath := filepath.Join(fullPathString)

	fmt.Println(fullPath)

	fileCreated, err := os.Create(localPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not create new file", err)
		return
	}

	_, err = io.Copy(fileCreated, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error copying info to new file", err)
		return
	}
	defer fileCreated.Close()

	video.ThumbnailURL = &fullPath

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not update video thumbnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
