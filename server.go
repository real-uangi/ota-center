package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const adminKeyHeader = "X-OTA-Admin-Key"

type Server struct {
	cfg   Config
	store *Store
}

type versionResponse struct {
	AppName     string `json:"app_name"`
	Version     string `json:"version"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	SHA256      string `json:"sha256"`
	DownloadURL string `json:"download_url"`
	UploadedAt  string `json:"uploaded_at"`
}

type uploadResponse struct {
	AppName    string `json:"app_name"`
	Version    string `json:"version"`
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	SHA256     string `json:"sha256"`
	UploadedAt string `json:"uploaded_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewServer(cfg Config, store *Store) *Server {
	return &Server{cfg: cfg, store: store}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/ota/upload", s.handleUpload)
	mux.HandleFunc("GET /ota/", s.handleOTA)
	return mux
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(adminKeyHeader) != s.cfg.AdminKey {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid admin key"})
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
		return
	}

	appName := r.FormValue("app_name")
	version := r.FormValue("version")
	if appName == "" || version == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "app_name and version are required"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer file.Close()

	metadata, err := s.store.SaveVersion(appName, version, file)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, uploadResponse{
		AppName:    appName,
		Version:    metadata.Version,
		FileName:   metadata.FileName,
		FileSize:   metadata.FileSize,
		SHA256:     metadata.SHA256,
		UploadedAt: metadata.UploadedAt,
	})
}

func (s *Server) handleOTA(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/ota/")
	parts := strings.Split(trimmed, "/")

	switch {
	case len(parts) == 3 && parts[1] == "versions" && parts[2] == "latest":
		s.handleLatestVersion(w, r, parts[0])
	case len(parts) == 3 && parts[1] == "versions":
		s.handleVersion(w, r, parts[0], parts[2])
	case len(parts) == 3 && parts[1] == "download":
		s.handleDownload(w, r, parts[0], parts[2])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleLatestVersion(w http.ResponseWriter, r *http.Request, appName string) {
	metadata, err := s.store.LatestVersion(appName)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, s.newVersionResponse(r, appName, metadata))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request, appName, version string) {
	metadata, err := s.store.GetVersionMetadata(appName, version)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, s.newVersionResponse(r, appName, metadata))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request, appName, version string) {
	file, metadata, err := s.store.OpenVersion(appName, version)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", metadata.FileName))
	http.ServeContent(w, r, metadata.FileName, parseUploadedAt(metadata.UploadedAt), file)
}

func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidAppName):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrVersionExists):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrAppNotFound), errors.Is(err, ErrVersionNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		if strings.Contains(err.Error(), "invalid version format") || strings.Contains(err.Error(), "parse revision time") {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (s *Server) downloadURL(r *http.Request, appName, version string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}

	return fmt.Sprintf("%s://%s/ota/%s/download/%s", scheme, r.Host, appName, version)
}

func (s *Server) newVersionResponse(r *http.Request, appName string, metadata VersionMetadata) versionResponse {
	return versionResponse{
		AppName:     appName,
		Version:     metadata.Version,
		FileName:    metadata.FileName,
		FileSize:    metadata.FileSize,
		SHA256:      metadata.SHA256,
		DownloadURL: s.downloadURL(r, appName, metadata.Version),
		UploadedAt:  metadata.UploadedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseUploadedAt(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}
