package main

import (
	"fmt"
	"io"
	"mime"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20

	// Store up to maxMemory bytes from that files
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		log.Println("Error parsing media type")
		return
	}

	log.Println(contentType)

	if contentType != "image/jpeg" && contentType != "image/png" {
		respondWithError(w, http.StatusInternalServerError,
			"Invalid mime type for thumbnail",
			err)
		return
	}


	video, err := cfg.db.GetVideo(videoID);
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to Get Video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unable to Authorize User", err)
		return
	}

	extension := strings.Split(contentType, "/")
	uniqueFileNameByte := make([]byte, 32)
	_, err = rand.Read(uniqueFileNameByte)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot create unique fileName", err)
		return
	}
	uniqueFileName := base64.RawURLEncoding.EncodeToString(uniqueFileNameByte)
	fileName := fmt.Sprintf("%s.%s", uniqueFileName, extension[1])
	uniquePath := filepath.Join(cfg.assetsRoot, fileName)
	log.Println(fileName)
	log.Println(uniquePath)
	newFile, err := os.Create(uniquePath)
	if err != nil {
		log.Println("Path does not exist")
		return;
	}

	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		log.Println("Error copying from multipart")
		return;
	}

	url := fmt.Sprintf("http://localhost:%s/%s",
		cfg.port, uniquePath)
	log.Println(url)
	video.VideoURL = &url

	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to Authorize User", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
