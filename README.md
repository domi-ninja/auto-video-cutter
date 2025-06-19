# Video Cutter

A command-line tool that analyzes gameplay videos to automatically detect excitement moments based on audio volume spikes. Perfect for streamers and content creators who want to quickly find highlight moments for editing.

## Features

- **Automatic excitement detection** - Analyzes audio volume patterns to find loud/exciting moments
- **LosslessCut integration** - Outputs CSV timestamps that can be imported directly into LosslessCut
- **Configurable thresholds** - Adjust sensitivity and minimum duration for detected moments
- **FFmpeg integration** - Supports all video formats that FFmpeg can handle

## Prerequisites

1. **FFmpeg** - Must be installed and available in your PATH
   - Windows: Download from https://ffmpeg.org/download.html
   - macOS: `brew install ffmpeg`
   - Linux: `sudo apt install ffmpeg` or equivalent

2. **Go 1.21+** - Required to build the application

## Installation

1. Clone or download this repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the application:
   ```bash
   go build -o video-cutter
   ```

## Usage

Basic usage:
```bash
./video-cutter -input gameplay.mp4
```

This will:
1. Extract audio from your video file
2. Analyze volume patterns to detect excitement
3. Generate a CSV file (e.g., `gameplay_markers.csv`) with timestamps

### Options

```bash
./video-cutter -input <video_file> [options]
```

**Options:**
- `-input` - Input video file path (required)
- `-output` - Output CSV file path (default: input_name_markers.csv)
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

## Using with LosslessCut

1. Run video-cutter on your video file
2. Open LosslessCut
3. Load your video file
4. Go to: **File → Import project → CSV segments**
5. Select the generated CSV file
6. Your excitement markers will appear as segments in the timeline

## How it Works

The tool uses a simple but effective algorithm:

1. **Audio Extraction** - Uses FFmpeg to extract mono audio at 44.1kHz
2. **Volume Analysis** - Calculates RMS volume in overlapping time windows
3. **Baseline Detection** - Establishes a baseline volume using median calculation
4. **Spike Detection** - Identifies periods where volume exceeds threshold × baseline
5. **Filtering** - Removes spikes shorter than minimum duration to avoid false positives

## Troubleshooting

**"ffmpeg not found"** - Make sure FFmpeg is installed and in your PATH

**"No excitement markers found"** - Try:
- Lowering the threshold (e.g., `-threshold 1.5`)
- Reducing minimum duration (e.g., `-min-duration 0.5`)
- Using `-verbose` to see analysis details

**"Not enough audio data"** - Your video might be too short or have very quiet audio

## CSV Output Format

The generated CSV file contains:
```csv
start_time,end_time,label
12.34,18.67,Excitement (2.3x)
45.12,52.89,Excitement (3.1x)
```

- `start_time`: Start timestamp in seconds
- `end_time`: End timestamp in seconds  
- `label`: Description with volume multiplier

This format is compatible with LosslessCut's CSV import feature. 