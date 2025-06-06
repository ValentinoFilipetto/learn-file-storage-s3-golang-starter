package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type MediaInfo struct {
           Streams []struct {
                Width  int `json:"width"`
                Height int `json:"height"`
            } `json:"streams"`
        }

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print-format", "json", "show_streams", filePath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
        return "", fmt.Errorf("ffprobe error: %w", err)
	}

	var commandOutput MediaInfo
	err = json.Unmarshal(stdout.Bytes(), &commandOutput)
	if err != nil {
	   return "", fmt.Errorf("Failed to unmarshal JSON: %w", err)
	}

	if len(commandOutput.Streams) == 0 {
		return "", fmt.Errorf("No streams found in media file: %v", err)
	}

	width := commandOutput.Streams[0].Width
	height := commandOutput.Streams[0].Height

	if width == 0 || height == 0 {
        return "", fmt.Errorf("Invalid width or height")
    }

	divisor := gcd(width, height)
	aspectW := width / divisor
	aspectH := height / divisor

	return fmt.Sprintf("%d:%d", aspectW, aspectH), nil
}

// gcd returns the greatest common divisor of a and b
func gcd(a, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}
