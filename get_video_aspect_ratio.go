package main

import (
   "os/exec"
   "bytes"
   "encoding/json"
   "log"
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
           log.Fatal(err)
	}

	var commandOutput MediaInfo
	err = json.Unmarshal(stdout.Bytes(), &commandOutput)
	if err != nil {
	   log.Fatal("Failed to unmarshal JSON: %v", err)
	}

	if len(commandOutpout.Streams) == 0 {
	    return "", fmt.Error("No streams found in media file: %v", err)
	}
        
	var width, height int
	width = commandOutput.Streams[0].Width
	height = commandOutput.Streams[0].Height
}
