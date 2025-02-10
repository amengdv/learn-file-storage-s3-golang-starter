package main

import (
	"io"
	"fmt"
	"net/http"

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
	r.ParseMultipartForm(maxMemory);

	file, header, err := r.FormFile("thumbnail");
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err);
		return
	}
	defer file.Close();

	contentType := header.Header.Get("Content-Type");
	data, err := io.ReadAll(file);
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read from file", err);
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

	th := thumbnail{
		data: data,
		mediaType: contentType,
	}
	
	videoThumbnails[videoID] = th

	url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)

	video.ThumbnailURL = &url

	if err = cfg.db.UpdateVideo(video); err != nil {
		delete(videoThumbnails, videoID)
		respondWithError(w, http.StatusUnauthorized, "Unable to Authorize User", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
