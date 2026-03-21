package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	jsonStruct := struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}{}
	err = json.Unmarshal(out.Bytes(), &jsonStruct)
	if err != nil {
		return "", err
	}
	if len(jsonStruct.Streams) == 0 {
		return "", fmt.Errorf("no streams found in video")
	}
	width := jsonStruct.Streams[0].Width
	height := jsonStruct.Streams[0].Height

	tolerance := 0.01 * float64(16*height)
	if math.Abs(float64(9*width-16*height)) <= tolerance {
		return "16:9", nil
	} else if math.Abs(float64(16*width-9*height)) <= tolerance {
		return "9:16", nil
	} else {
		return "other", nil
	}
}
