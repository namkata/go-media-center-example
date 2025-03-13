package utils

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MediaMetadata holds technical details about media files
type MediaMetadata struct {
	// Common metadata
	FileType   string      `json:"file_type"`
	MimeType   string      `json:"mime_type"`
	Size       int64       `json:"size"`
	UploadedAt string      `json:"uploaded_at"`
	Dimensions *Dimensions `json:"dimensions,omitempty"`
	Format     string      `json:"format"`

	// Image specific metadata
	ColorSpace  string `json:"color_space,omitempty"`
	ColorDepth  int    `json:"color_depth,omitempty"`
	HasAlpha    bool   `json:"has_alpha,omitempty"`
	Orientation string `json:"orientation,omitempty"`

	// Video specific metadata
	Duration    string `json:"duration,omitempty"`
	Bitrate     string `json:"bitrate,omitempty"`
	VideoCodec  string `json:"video_codec,omitempty"`
	AudioCodec  string `json:"audio_codec,omitempty"`
	FrameRate   string `json:"frame_rate,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
}

// Dimensions holds width and height information
type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ExtractMetadata extracts metadata from a media file
func ExtractMetadata(file *multipart.FileHeader) (*MediaMetadata, error) {
	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	// Read the first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %v", err)
	}

	// Reset file pointer
	f.Seek(0, 0)

	contentType := GetMimeType(buffer)
	metadata := &MediaMetadata{
		FileType:   GetFileType(file.Filename),
		MimeType:   contentType,
		Size:       file.Size,
		UploadedAt: time.Now().Format(time.RFC3339),
		Format:     strings.TrimPrefix(filepath.Ext(file.Filename), "."),
	}

	// Extract specific metadata based on file type
	switch {
	case strings.HasPrefix(contentType, "image/"):
		if err := extractImageMetadata(f, metadata); err != nil {
			return nil, fmt.Errorf("failed to extract image metadata: %v", err)
		}
	case strings.HasPrefix(contentType, "video/"):
		if err := extractVideoMetadata(f, metadata); err != nil {
			return nil, fmt.Errorf("failed to extract video metadata: %v", err)
		}
	}

	return metadata, nil
}

// extractImageMetadata extracts metadata specific to images
func extractImageMetadata(f multipart.File, metadata *MediaMetadata) error {
	// Decode image for dimensions and color info
	img, format, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := img.Bounds()
	metadata.Dimensions = &Dimensions{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}

	// Set orientation
	if metadata.Dimensions.Width > metadata.Dimensions.Height {
		metadata.Orientation = "landscape"
	} else if metadata.Dimensions.Width < metadata.Dimensions.Height {
		metadata.Orientation = "portrait"
	} else {
		metadata.Orientation = "square"
	}

	// Get color model information
	switch format {
	case "jpeg":
		metadata.ColorSpace = "RGB"
		metadata.ColorDepth = 24
	case "png":
		metadata.ColorSpace = "RGB"
		metadata.ColorDepth = 32
		metadata.HasAlpha = true
	case "gif":
		metadata.ColorSpace = "RGB"
		metadata.ColorDepth = 8
		metadata.HasAlpha = true
	}

	return nil
}

// extractVideoMetadata extracts metadata specific to videos using ffprobe
func extractVideoMetadata(f multipart.File, metadata *MediaMetadata) error {
	// Create a temporary file for FFmpeg to process
	tempFile, err := SaveTempFile(f)
	if err != nil {
		return fmt.Errorf("failed to save temporary file: %v", err)
	}
	defer os.Remove(tempFile)

	// Use ffprobe to get video metadata
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		tempFile)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to extract video metadata: %v", err)
	}

	// Parse the JSON output
	var result struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width,omitempty"`
			Height     int    `json:"height,omitempty"`
			RFrameRate string `json:"r_frame_rate,omitempty"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse video metadata: %v", err)
	}

	// Extract video stream information
	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			metadata.VideoCodec = stream.CodecName
			metadata.Dimensions = &Dimensions{
				Width:  stream.Width,
				Height: stream.Height,
			}
			// Parse frame rate (usually in "num/den" format)
			if parts := strings.Split(stream.RFrameRate, "/"); len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den > 0 {
					metadata.FrameRate = fmt.Sprintf("%.2f", num/den)
				}
			}
			// Calculate aspect ratio
			if stream.Width > 0 && stream.Height > 0 {
				metadata.AspectRatio = fmt.Sprintf("%d:%d", stream.Width, stream.Height)
			}
		case "audio":
			metadata.AudioCodec = stream.CodecName
		}
	}

	// Set format information
	metadata.Duration = result.Format.Duration
	metadata.Bitrate = result.Format.BitRate

	return nil
}

// SaveTempFile saves a multipart.File to a temporary file
func SaveTempFile(f multipart.File) (string, error) {
	tempFile, err := os.CreateTemp("", "media-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, f)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %v", err)
	}

	return tempFile.Name(), nil
}

// GetMimeType determines the MIME type from file contents
func GetMimeType(buffer []byte) string {
	return http.DetectContentType(buffer)
}

func DetectMimeType(resp *http.Response, file io.ReadSeeker, filename string) string {
	// 1. Try detecting from file content
	buffer := make([]byte, 512)
	file.Seek(0, 0) // Reset reader before reading
	_, err := file.Read(buffer)
	file.Seek(0, 0) // Reset after reading

	if err == nil {
		mimeType := http.DetectContentType(buffer)
		if mimeType != "application/octet-stream" { // Not generic binary
			return mimeType
		}
	}

	// 2. Try Content-Type from response headers
	if resp != nil {
		mimeType := resp.Header.Get("Content-Type")
		if mimeType != "" {
			return mimeType
		}
	}

	// 3. Infer from file extension
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
	}

	if mimeType, found := mimeTypes[ext]; found {
		return mimeType
	}

	return "application/octet-stream" // Default fallback
}
