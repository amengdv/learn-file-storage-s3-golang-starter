package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func processVideoForFastStart(filePath string) (string, error) {
	fileName := strings.Split(filePath, ".")
	newFilePath := fmt.Sprintf("%s.processing.mp4", fileName[0])

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags",
		"faststart", "-f", "mp4", newFilePath)

	log.Printf("Defining command, filepath is : %v\n", filePath)
	log.Printf("Defining command, newFilePath is : %v\n", newFilePath)

	if err := cmd.Run(); err != nil {
		log.Println(err)
		return "", err
	}

	return newFilePath, nil
}

func getVideoAspectRatio(filePath string) (string, error) {

	commandName := "ffprobe"

	cmd := exec.Command(commandName, "-v", "error", "-print_format", "json", "-show_streams", filePath)
	log.Printf("Defining command filepath is : %v\n", filePath)

	buffer := bytes.Buffer{}
	cmd.Stdout = &buffer

	if err := cmd.Run(); err != nil {
		log.Println(err)
		return "", err
	}

	data := buffer.Bytes()

	type Streams struct {
		Stream []struct{
			Width int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	payload := Streams{}

	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Error unmarshal json for ffprobe: %v", err)
		return "", err
	}

	log.Printf("Payload: %v\n", payload)

	ratioCalc := payload.Stream[0].Width / payload.Stream[0].Height

	ratio := "other"

	toleranceRange := 0.5
	floatRatio := float64(ratioCalc) - float64(16/9)
	if  floatRatio < 0 {
		ratio = "9:16"
	} else if floatRatio >= 0 && floatRatio <= toleranceRange {
		ratio = "16:9"
	}

	log.Printf("File Ratio: %v", ratio)
	return ratio, nil
}
