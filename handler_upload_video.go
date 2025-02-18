package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	const uploadLimit = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to Get Video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unable to Authorize User", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		log.Println("Error parsing media type")
		respondWithError(w, http.StatusInternalServerError, "Unable to parse form file", err)
		return
	}

	if contentType != "video/mp4" {
		log.Println("Invalid Mime Type for video")
		respondWithError(w, http.StatusInternalServerError,
			"Invalid mime type for video",
			err)
		return
	}

	tmpFileName := "tubely-upload.mp4"
	tmpVideoFile, err := os.CreateTemp("", tmpFileName)
	if err != nil {
		log.Println("Cannot create temporary file for video")
		respondWithError(w, http.StatusInternalServerError,
			"Cannot create temporary file for video",
			err)
		return
	}

	defer os.Remove(tmpFileName)
	defer tmpVideoFile.Close()

	_, err = io.Copy(tmpVideoFile, file)
	if err != nil {
		log.Println("Error copying from body to tmpVideoFile")
		respondWithError(w, http.StatusInternalServerError,
			"Error copying from body to tmpVideoFile",
			err)
		return;
	}

	_, err = tmpVideoFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			"failed to read video file from start",
			err)
		return;
	}

	outputFastProcessVideo, err := processVideoForFastStart(tmpVideoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot Preprocess Video", err)
		return
	}

	preprocessVideo, err := os.OpenFile(outputFastProcessVideo, os.O_RDONLY, os.ModePerm)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot Open Preprocessed Video", err)
		return
	}

	defer os.Remove(preprocessVideo.Name())
	defer preprocessVideo.Close()

	uniqueFileNameByte := make([]byte, 32)
	_, err = rand.Read(uniqueFileNameByte)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot create unique fileName", err)
		return
	}

	ratio, err := getVideoAspectRatio(tmpVideoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot get aspect video ratio", err)
		return
	}
	
	prefix := "other"
	if ratio == "16:9" {
		prefix = "landscape"
	} else if ratio == "9:16" {
		prefix = "portrait"
	}

	uniqueFileName := base64.RawURLEncoding.EncodeToString(uniqueFileNameByte)
	key := fmt.Sprintf("%s/%s.mp4", prefix, uniqueFileName)
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(key),
		Body: preprocessVideo,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			"Failed to put object to S3",
			err)
		return;
	}

	url := 	fmt.Sprintf("%s/%s", cfg.s3CfDistribution,key)
	log.Println(url)
	video.VideoURL = &url

	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to Authorize User", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

