package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
)

type UploadRequest struct {
	FileName     string `json:"fileName"`
	FileMimeType string `json:"fileMimeType"`
	Content      string `json:"content"`
}

type UploadResponse struct {
	FileMetadataID string `json:"fileMetadataId"`
}

// 上传图片到 Grok 服务器，返回 fileMetadataId
func UploadImage(imageURL, cookie string) (string, error) {
	isURL := strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://")
	var imageBuffer, mimeType string

	if isURL {
		// 使用 TLS 客户端下载图片
		client := GetHTTPClient()
		req, err := fhttp.NewRequest("GET", imageURL, nil)
		if err != nil {
			return "", err
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		imageBuffer = base64.StdEncoding.EncodeToString(data)
		mimeType = resp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
	} else {
		// Base64 格式: data:image/jpeg;base64,xxx
		if strings.HasPrefix(imageURL, "data:") {
			parts := strings.SplitN(imageURL, ",", 2)
			if len(parts) == 2 {
				imageBuffer = parts[1]
				if strings.Contains(parts[0], ";") {
					mimeType = strings.Split(parts[0], ";")[0][5:]
				}
			}
		} else {
			imageBuffer = imageURL
		}
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
	}

	ext := "jpg"
	if strings.Contains(mimeType, "/") {
		ext = strings.Split(mimeType, "/")[1]
	}
	fileName := fmt.Sprintf("image.%s", ext)

	uploadReq := UploadRequest{
		FileName:     fileName,
		FileMimeType: mimeType,
		Content:      imageBuffer,
	}

	body, err := json.Marshal(uploadReq)
	if err != nil {
		return "", err
	}

	req, err := fhttp.NewRequest("POST", BaseURL+"/rest/app-chat/upload-file", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	SetUploadHeaders(req, cookie)

	client := GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText := string(bodyBytes)
		LogError("Upload image failed - Status: %d, Response: %s", resp.StatusCode, bodyText)
		return "", fmt.Errorf("upload image failed: status %d", resp.StatusCode)
	}

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	return uploadResp.FileMetadataID, nil
}

// 从消息中提取并上传所有图片
func ExtractAndUploadImages(messages []Message, cookie string) ([]string, error) {
	var imageURLs []string
	for _, msg := range messages {
		_, urls := msg.ParseContent()
		imageURLs = append(imageURLs, urls...)
	}

	if len(imageURLs) == 0 {
		return nil, nil
	}

	var fileIDs []string
	for _, url := range imageURLs {
		fileID, err := UploadImage(url, cookie)
		if err != nil {
			LogError("Failed to upload image: %v", err)
			continue
		}
		if fileID != "" {
			fileIDs = append(fileIDs, fileID)
		}
	}

	return fileIDs, nil
}
