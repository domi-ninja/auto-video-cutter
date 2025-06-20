# Vibe coded: Auto Video Cutter 

A command-line tool that uses a crude heuristic to attempt to automatically detect excitement moments in a video, solely based on audio volume spikes. I'm trying to quickly find highlight moments without doing anything.

## Features

- **Automatic excitement detection** - Analyzes audio volume patterns to find loud/exciting moments
- **LosslessCut integration** - Outputs JSON project files that can be opened directly in LosslessCut
- **Auto adjusted thresholds** - Adjust sensitivity and minimum duration for detected moments
- **FFmpeg integration** - Supports all video formats that FFmpeg can handle

## Prerequisites

1. **FFmpeg** - Must be installed and available in your PATH
2. **Go 1.21+** - Required to build the application

## Usage

Basic usage:
```bash
./video-cutter -input gameplay.mp4
```

This will:
1. Extract audio from your video file
2. Analyze volume patterns to detect excitement compared to average
3. Generate a LosslessCut project file (e.g., `FILENAME-proj.llc`) with cut segments

### Options

```bash
./video-cutter -input <video_file> [options]
```

**Options:**
- `-input` - Input video file path (required)
- `-output` - Output LosslessCut project file path (default: input_name-proj.llc)
- `-threshold` - Volume spike threshold multiplier (default: 2.0)
- `-min-duration` - Minimum excitement duration in seconds (default: 1.0)
- `-window` - Analysis window size in milliseconds (default: 1000)
- `-verbose` - Enable verbose logging

### Examples

```bash
# Basic usage
./video-cutter -input my_stream.mp4

# More sensitive detection (lower threshold)
./video-cutter -input my_stream.mp4 -threshold 1.5

# Only detect longer excitement periods
./video-cutter -input my_stream.mp4 -min-duration 3.0

# Enable verbose logging to see detection details
./video-cutter -input my_stream.mp4 -verbose
```

## Output Format

The tool generates LosslessCut project files in JSON format:

```json
{
  "version": 1,
  "mediaFileName": "FILENAME.mp4",
  "cutSegments": [
    {
      "start": 120.45,
      "end": 180.45,
      "name": "Excitement (2.3x)"
    },
    {
      "start": 250.12,
      "end": 310.12,
      "name": "Excitement (1.8x)"
    }
  ]
}
```