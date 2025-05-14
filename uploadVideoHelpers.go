package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

type videoDimensions struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error running command Error: %v\n", err)
		return "", err
	}

	var data videoDimensions

	err = json.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		fmt.Printf("err unmarshalling data Error: %v\n", err)
		return "", err
	}
	videoHeight := 0
	videoWidth := 0
	if len(data.Streams) > 0 {
		for _, video := range data.Streams {
			if video.Height != 0 && video.Width != 0 {
				videoHeight = video.Height
				videoWidth = video.Width
				break
			}
		}
		if videoHeight == 0 || videoWidth == 0 {
			fmt.Printf("Could not get the dimensions of the video")
			return "", fmt.Errorf("could not find the dimensions")
		}

		if float64(videoWidth)/float64(videoHeight) >= 1.7 && float64(videoWidth)/float64(videoHeight) <= 1.8 {
			return "16:9", nil

		} else if float64(videoWidth)/float64(videoHeight) >= .56 && float64(videoWidth)/float64(videoHeight) <= .6 {
			return "9:16", nil
		}
	}
	return "other", nil
}

func processVideoForFastStart(filepath string) (string, error) {
	newFilePath := fmt.Sprintf("%v.processing", filepath)
	cmd := exec.Command("ffmpeg", "-i", filepath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Could not run the command Error: %v\n", err)
		return "", err
	}
	return newFilePath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	v4Req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		fmt.Printf("error with the presignGetObject function Error: %v", err)
		return "", err
	}
	return v4Req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil || *video.VideoURL == "" {
		fmt.Println("there is no url")
		return video, nil
	}
	if strings.HasPrefix(*video.VideoURL, "http") {
		return video, nil
	}

	splitted := strings.Split(*video.VideoURL, ",")
	if len(splitted) < 2 {
		fmt.Printf("incorrect size, do not have bucket and key Size: %v  input: %v\n", len(splitted), splitted[0])
		//return video, fmt.Errorf("bad format for videoURL")
		return video, nil
	}
	bucket := splitted[0]
	s3key := splitted[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, s3key, time.Minute*15)
	if err != nil {
		fmt.Printf("error getting the presignedURL Error: %v", err)
		return video, err
	}
	video.VideoURL = &presignedURL
	return video, nil
}
