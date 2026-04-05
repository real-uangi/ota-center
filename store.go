package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var appNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var (
	ErrAppNotFound     = errors.New("app not found")
	ErrVersionNotFound = errors.New("version not found")
	ErrVersionExists   = errors.New("version already exists")
	ErrInvalidAppName  = errors.New("invalid app name")
)

type VersionMetadata struct {
	Version    string `json:"version"`
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	SHA256     string `json:"sha256"`
	UploadedAt string `json:"uploaded_at"`
}

type Store struct {
	dataDir string
	mu      sync.Mutex
}

func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

func (s *Store) SaveVersion(appName, version string, body io.Reader) (VersionMetadata, error) {
	if !appNamePattern.MatchString(appName) {
		return VersionMetadata{}, ErrInvalidAppName
	}

	if _, err := ParseVersion(version); err != nil {
		return VersionMetadata{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.loadIndexLocked(appName)
	if err != nil && !errors.Is(err, ErrAppNotFound) {
		return VersionMetadata{}, err
	}

	for _, item := range versions {
		if item.Version == version {
			return VersionMetadata{}, ErrVersionExists
		}
	}

	appDir := s.appDir(appName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return VersionMetadata{}, fmt.Errorf("create app dir: %w", err)
	}

	fileName := version + ".bin"
	filePath := filepath.Join(appDir, fileName)
	tempPath := filePath + ".tmp"
	if _, err := os.Stat(filePath); err == nil {
		return VersionMetadata{}, ErrVersionExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return VersionMetadata{}, fmt.Errorf("check existing file: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("create temp file: %w", err)
	}

	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return VersionMetadata{}, fmt.Errorf("save file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return VersionMetadata{}, fmt.Errorf("close temp file: %w", closeErr)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return VersionMetadata{}, fmt.Errorf("move temp file: %w", err)
	}

	metadata := VersionMetadata{
		Version:    version,
		FileName:   fileName,
		FileSize:   size,
		SHA256:     hex.EncodeToString(hash.Sum(nil)),
		UploadedAt: time.Now().UTC().Format(time.RFC3339),
	}

	versions = append(versions, metadata)
	if err := s.saveIndexLocked(appName, versions); err != nil {
		_ = os.Remove(filePath)
		return VersionMetadata{}, err
	}

	return metadata, nil
}

func (s *Store) LatestVersion(appName string) (VersionMetadata, error) {
	if !appNamePattern.MatchString(appName) {
		return VersionMetadata{}, ErrInvalidAppName
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.loadIndexLocked(appName)
	if err != nil {
		return VersionMetadata{}, err
	}
	if len(versions) == 0 {
		return VersionMetadata{}, ErrAppNotFound
	}

	current := versions[0]
	currentVersion, err := ParseVersion(current.Version)
	if err != nil {
		return VersionMetadata{}, err
	}

	for _, item := range versions[1:] {
		parsed, err := ParseVersion(item.Version)
		if err != nil {
			return VersionMetadata{}, err
		}
		if CompareVersion(parsed, currentVersion) > 0 {
			current = item
			currentVersion = parsed
		}
	}

	return current, nil
}

func (s *Store) OpenVersion(appName, version string) (*os.File, VersionMetadata, error) {
	if !appNamePattern.MatchString(appName) {
		return nil, VersionMetadata{}, ErrInvalidAppName
	}

	if _, err := ParseVersion(version); err != nil {
		return nil, VersionMetadata{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.loadIndexLocked(appName)
	if err != nil {
		return nil, VersionMetadata{}, err
	}

	for _, item := range versions {
		if item.Version != version {
			continue
		}

		filePath := filepath.Join(s.appDir(appName), item.FileName)
		file, err := os.Open(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, VersionMetadata{}, ErrVersionNotFound
			}
			return nil, VersionMetadata{}, fmt.Errorf("open version file: %w", err)
		}

		return file, item, nil
	}

	return nil, VersionMetadata{}, ErrVersionNotFound
}

func (s *Store) GetVersionMetadata(appName, version string) (VersionMetadata, error) {
	if !appNamePattern.MatchString(appName) {
		return VersionMetadata{}, ErrInvalidAppName
	}

	if _, err := ParseVersion(version); err != nil {
		return VersionMetadata{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.loadIndexLocked(appName)
	if err != nil {
		return VersionMetadata{}, err
	}

	for _, item := range versions {
		if item.Version == version {
			return item, nil
		}
	}

	return VersionMetadata{}, ErrVersionNotFound
}

func (s *Store) loadIndexLocked(appName string) ([]VersionMetadata, error) {
	data, err := os.ReadFile(s.indexPath(appName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrAppNotFound
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	var versions []VersionMetadata
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	return versions, nil
}

func (s *Store) saveIndexLocked(appName string, versions []VersionMetadata) error {
	data, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	indexPath := s.indexPath(appName)
	tempPath := indexPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp index: %w", err)
	}

	if err := os.Rename(tempPath, indexPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("move temp index: %w", err)
	}

	return nil
}

func (s *Store) appDir(appName string) string {
	return filepath.Join(s.dataDir, appName)
}

func (s *Store) indexPath(appName string) string {
	return filepath.Join(s.appDir(appName), "index.json")
}
