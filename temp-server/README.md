# Temporary Streaming HTTP Server

A Go-based HTTP server with comprehensive streaming capabilities.

## Features

- **Server-Sent Events (SSE)**: Real-time event streaming
- **Chunked Transfer Encoding**: Stream data in chunks
- **File Streaming**: Stream any file with range request support
- **Media Streaming**: Optimized for video/audio with range support
- **Data Streaming**: Continuous data streaming
- **Web Interface**: Built-in testing interface

## Quick Start

```bash
cd temp-server
go run main.go
```

The server will start on `http://localhost:8080`

## Endpoints

### Web Interface
- `GET /` - Interactive web interface for testing all streaming endpoints

### Streaming Endpoints
- `GET /stream/events` - Server-Sent Events (real-time updates)
- `GET /stream/chunked` - Chunked transfer encoding demo
- `GET /stream/data` - Continuous data streaming
- `GET /stream/file/<filename>` - Stream files from parent directory with range support
- `GET /media/<filename>` - Media streaming with range support for video/audio

## Usage Examples

### Stream a video file
```bash
curl http://localhost:8080/media/video.mp4
```

### Test Server-Sent Events
```bash
curl http://localhost:8080/stream/events
```

### Stream with range requests
```bash
curl -H "Range: bytes=0-1000" http://localhost:8080/stream/file/document.pdf
```

## Environment Variables

- `PORT` - Server port (default: 8080)

## File Access

The server can stream files from the parent directory (where your video files are located) for security. File paths are sanitized to prevent directory traversal attacks.

## Notes

- Supports HTTP range requests for efficient video streaming
- Handles client disconnections gracefully
- CORS enabled for cross-origin requests
- Automatic content-type detection based on file extensions 