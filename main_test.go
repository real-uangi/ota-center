package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseVersion(t *testing.T) {
	if _, err := ParseVersion("v1.2-202604051230"); err != nil {
		t.Fatalf("expected version to be valid: %v", err)
	}

	if _, err := ParseVersion("v1.2.3-202604051230"); err == nil {
		t.Fatal("expected patch version to be invalid")
	}

	if _, err := ParseVersion("v1.2-20260405"); err == nil {
		t.Fatal("expected short timestamp to be invalid")
	}
}

func TestCompareVersion(t *testing.T) {
	left, _ := ParseVersion("v1.3-202604011000")
	right, _ := ParseVersion("v1.2-202604051000")
	if CompareVersion(left, right) <= 0 {
		t.Fatal("expected major/minor comparison to win")
	}

	newer, _ := ParseVersion("v1.2-202604051230")
	older, _ := ParseVersion("v1.2-202604041230")
	if CompareVersion(newer, older) <= 0 {
		t.Fatal("expected revision time comparison to win")
	}
}

func TestUploadLatestAndDownloadFlow(t *testing.T) {
	dataDir := t.TempDir()
	server := httptest.NewServer(NewServer(Config{
		Port:     "8080",
		DataDir:  dataDir,
		AdminKey: "secret",
	}, NewStore(dataDir)).routes())
	defer server.Close()

	status, body := uploadBinary(t, server.URL, "secret", "demo-app", "v1.2-202604051230", []byte("firmware-v1"))
	if status != http.StatusCreated {
		t.Fatalf("expected upload success, got status=%d body=%s", status, body)
	}

	status, body = uploadBinary(t, server.URL, "secret", "demo-app", "v1.3-202604011000", []byte("firmware-v2"))
	if status != http.StatusCreated {
		t.Fatalf("expected second upload success, got status=%d body=%s", status, body)
	}

	latestResp, err := http.Get(server.URL + "/ota/demo-app/latest_version")
	if err != nil {
		t.Fatalf("latest_version request failed: %v", err)
	}
	defer latestResp.Body.Close()

	if latestResp.StatusCode != http.StatusOK {
		t.Fatalf("expected latest_version success, got %d", latestResp.StatusCode)
	}

	var latest latestVersionResponse
	if err := json.NewDecoder(latestResp.Body).Decode(&latest); err != nil {
		t.Fatalf("decode latest_version response: %v", err)
	}

	if latest.Version != "v1.3-202604011000" {
		t.Fatalf("expected latest version to be v1.3-202604011000, got %s", latest.Version)
	}

	downloadResp, err := http.Get(latest.DownloadURL)
	if err != nil {
		t.Fatalf("download request failed: %v", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("expected download success, got %d", downloadResp.StatusCode)
	}

	content := new(bytes.Buffer)
	if _, err := content.ReadFrom(downloadResp.Body); err != nil {
		t.Fatalf("read download body: %v", err)
	}

	if content.String() != "firmware-v2" {
		t.Fatalf("unexpected downloaded content: %s", content.String())
	}
}

func TestUploadAuthAndDuplicateVersion(t *testing.T) {
	dataDir := t.TempDir()
	server := httptest.NewServer(NewServer(Config{
		Port:     "8080",
		DataDir:  dataDir,
		AdminKey: "secret",
	}, NewStore(dataDir)).routes())
	defer server.Close()

	status, _ := uploadBinary(t, server.URL, "wrong", "demo-app", "v1.2-202604051230", []byte("firmware"))
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized upload, got %d", status)
	}

	status, body := uploadBinary(t, server.URL, "secret", "demo-app", "v1.2-202604051230", []byte("firmware"))
	if status != http.StatusCreated {
		t.Fatalf("expected first upload success, got status=%d body=%s", status, body)
	}

	status, _ = uploadBinary(t, server.URL, "secret", "demo-app", "v1.2-202604051230", []byte("firmware"))
	if status != http.StatusConflict {
		t.Fatalf("expected duplicate upload conflict, got %d", status)
	}
}

func TestNotFoundCases(t *testing.T) {
	dataDir := t.TempDir()
	server := httptest.NewServer(NewServer(Config{
		Port:     "8080",
		DataDir:  dataDir,
		AdminKey: "secret",
	}, NewStore(dataDir)).routes())
	defer server.Close()

	latestResp, err := http.Get(server.URL + "/ota/unknown/latest_version")
	if err != nil {
		t.Fatalf("latest_version request failed: %v", err)
	}
	defer latestResp.Body.Close()

	if latestResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected latest_version 404, got %d", latestResp.StatusCode)
	}

	downloadResp, err := http.Get(server.URL + "/ota/unknown/download/v1.2-202604051230")
	if err != nil {
		t.Fatalf("download request failed: %v", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected download 404, got %d", downloadResp.StatusCode)
	}
}

func uploadBinary(t *testing.T, baseURL, key, appName, version string, content []byte) (int, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("app_name", appName); err != nil {
		t.Fatalf("write app_name field: %v", err)
	}
	if err := writer.WriteField("version", version); err != nil {
		t.Fatalf("write version field: %v", err)
	}

	fileWriter, err := writer.CreateFormFile("file", "firmware.bin")
	if err != nil {
		t.Fatalf("create file field: %v", err)
	}
	if _, err := fileWriter.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/admin/ota/upload", &body)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	req.Header.Set(adminKeyHeader, key)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("run upload request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read upload response: %v", err)
	}

	return resp.StatusCode, string(responseBody)
}
