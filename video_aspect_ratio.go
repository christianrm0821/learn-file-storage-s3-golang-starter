package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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
