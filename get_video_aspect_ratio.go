package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type MediaInfo struct {
           Streams []struct {
                Width  int `json:"width"`
                Height int `json:"height"`
            } `json:"streams"`
        }

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
        return "", fmt.Errorf("ffprobe error: %w", err)
	}

	var commandOutput MediaInfo
	err = json.Unmarshal(stdout.Bytes(), &commandOutput)
	if err != nil {
	   return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if len(commandOutput.Streams) == 0 {
		return "", fmt.Errorf("no streams found in media file: %v", err)
	}

	width := commandOutput.Streams[0].Width
	height := commandOutput.Streams[0].Height

	if width == 0 || height == 0 {
        return "", fmt.Errorf("invalid width or height")
    }

	aspectRatio := float64(width) / float64(height)
	tolerance := 0.05

	if math.Abs(aspectRatio - 16.0 / 9.0) < tolerance {
		return "16:9", nil
	} else if math.Abs(aspectRatio - 9.0 / 16.0) < tolerance {
		return "9:16", nil
	}

	return "other", nil
}