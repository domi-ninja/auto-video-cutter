package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// Serve files from parent directory
	parentDir := filepath.Join("..", ".")
	absPath, err := filepath.Abs(parentDir)
	if err != nil {
		log.Fatal("Failed to get absolute path:", err)
	}

	log.Printf("Starting file server on port %s", port)
	log.Printf("Serving files from: %s", absPath)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			listFiles(w, r, absPath)
			return
		}

		filePath := filepath.Join(absPath, strings.TrimPrefix(r.URL.Path, "/"))
		serveFileWithRangeSupport(w, r, filePath)
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func listFiles(w http.ResponseWriter, r *http.Request, dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
    <title>File Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .file-list { list-style: none; padding: 0; }
        .file-item { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .file-item a { text-decoration: none; color: #0066cc; font-weight: bold; }
        .file-item a:hover { text-decoration: underline; }
        .file-size { color: #666; font-size: 0.9em; }
        .video-file { background-color: #f0f8ff; }
    </style>
</head>
<body>
    <h1>Available Files</h1>
    <ul class="file-list">
`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		info, _ := file.Info()
		size := formatFileSize(info.Size())

		isVideo := strings.HasSuffix(strings.ToLower(name), ".mp4") ||
			strings.HasSuffix(strings.ToLower(name), ".mov") ||
			strings.HasSuffix(strings.ToLower(name), ".avi") ||
			strings.HasSuffix(strings.ToLower(name), ".mkv")

		cssClass := ""
		if isVideo {
			cssClass = "video-file"
		}

		fmt.Fprintf(w, `
        <li class="file-item %s">
            <a href="/%s">%s</a>
            <span class="file-size">(%s)</span>
        </li>`, cssClass, name, name, size)
	}

	fmt.Fprint(w, `
    </ul>
</body>
</html>
`)
}

func serveFileWithRangeSupport(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to open file", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		http.Error(w, "Cannot serve directory", http.StatusBadRequest)
		return
	}

	fileSize := stat.Size()
	fileName := filepath.Base(filePath)

	// Set content type based on file extension
	contentType := getContentType(fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))

	// Handle range requests
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// No range request, serve entire file
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		http.ServeContent(w, r, fileName, stat.ModTime(), file)
		return
	}

	// Parse range header
	ranges, err := parseRange(rangeHeader, fileSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if len(ranges) == 1 {
		// Single range request
		start, end := ranges[0].start, ranges[0].end

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)

		_, err = file.Seek(start, 0)
		if err != nil {
			log.Printf("Failed to seek file: %v", err)
			return
		}

		_, err = io.CopyN(w, file, end-start+1)
		if err != nil {
			log.Printf("Failed to copy file range: %v", err)
		}
	} else {
		// Multiple ranges not supported for simplicity
		http.Error(w, "Multiple ranges not supported", http.StatusRequestedRangeNotSatisfiable)
	}
}

type httpRange struct {
	start, end int64
}

func parseRange(rangeHeader string, fileSize int64) ([]httpRange, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("invalid range header")
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, ",")

	var ranges []httpRange
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "-") {
			// Suffix range
			suffix, err := strconv.ParseInt(part[1:], 10, 64)
			if err != nil {
				return nil, err
			}
			start := fileSize - suffix
			if start < 0 {
				start = 0
			}
			ranges = append(ranges, httpRange{start, fileSize - 1})
		} else if strings.HasSuffix(part, "-") {
			// Prefix range
			start, err := strconv.ParseInt(part[:len(part)-1], 10, 64)
			if err != nil {
				return nil, err
			}
			ranges = append(ranges, httpRange{start, fileSize - 1})
		} else {
			// Full range
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format")
			}

			start, err := strconv.ParseInt(rangeParts[0], 10, 64)
			if err != nil {
				return nil, err
			}

			end, err := strconv.ParseInt(rangeParts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if start > end || end >= fileSize {
				return nil, fmt.Errorf("invalid range values")
			}

			ranges = append(ranges, httpRange{start, end})
		}
	}

	return ranges, nil
}

func getContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
		return "video/webm"
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	if size >= GB {
		return fmt.Sprintf("%.1fGB", float64(size)/GB)
	} else if size >= MB {
		return fmt.Sprintf("%.1fMB", float64(size)/MB)
	} else if size >= KB {
		return fmt.Sprintf("%.1fKB", float64(size)/KB)
	}
	return fmt.Sprintf("%dB", size)
}
