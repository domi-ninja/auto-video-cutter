# Video Cutter

A command-line tool that analyzes gameplay videos to automatically detect excitement moments based on audio volume spikes. Perfect for streamers and content creators who want to quickly find highlight moments for editing.

## Features

- **Automatic excitement detection** - Analyzes audio volume patterns to find loud/exciting moments
- **LosslessCut integration** - Outputs JSON project files that can be opened directly in LosslessCut
- **Smart segment extension** - Excitement markers over 1.0x automatically include the next minute of content
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
3. Generate a LosslessCut project file (e.g., `gameplay_markers.proj.llc`) with cut segments
4. **For excitement markers with score > 1.0x, automatically extend to include the next minute**

### Options

```bash
./video-cutter -input <video_file> [options]
```

**Options:**
- `-input` - Input video file path (required)
- `-output` - Output LosslessCut project file path (default: input_name_markers.proj.llc)
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
3. Go to: **File → Open** and select the generated `.proj.llc` file
4. Your video and excitement markers will load automatically
5. All segments are ready for export or further editing

## How it Works

The tool uses an enhanced algorithm:

1. **Audio Extraction** - Uses FFmpeg to extract mono audio at 44.1kHz
2. **Volume Analysis** - Calculates RMS volume in overlapping time windows
3. **Baseline Detection** - Establishes a baseline volume using median calculation
4. **Peak Tracking** - Tracks peak volume during each excitement period
5. **Spike Detection** - Identifies periods where volume exceeds threshold × baseline
6. **Smart Extension** - For excitement scores > 1.0x, extends segment to include next 60 seconds
7. **Filtering** - Removes spikes shorter than minimum duration to avoid false positives

## Output Format

The tool generates LosslessCut project files in JSON format:

```json
{
  "version": 1,
  "mediaFileName": "gameplay.mp4",
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

**Key Features:**
- Segments with excitement score > 1.0x are automatically extended to 60 seconds
- Segments won't extend beyond the video duration
- Each segment includes the excitement multiplier in the name

## Troubleshooting

**"ffmpeg not found"** - Make sure FFmpeg is installed and in your PATH

**"No excitement markers found"** - Try:
- Lowering the threshold (e.g., `-threshold 1.5`)
- Reducing minimum duration (e.g., `-min-duration 0.5`)
- Using `-verbose` to see analysis details

**"Not enough audio data"** - Your video might be too short or have very quiet audio

**Segments too long/short** - The 1-minute extension only applies to excitement scores > 1.0x. Adjust the `-threshold` parameter to control which segments get extended.

## Changelog

### v2.0 - LosslessCut Integration
- ✅ Changed output from CSV to LosslessCut JSON format
- ✅ Added smart segment extension for high-excitement moments (>1.0x score)
- ✅ Improved peak volume tracking during excitement periods
- ✅ Direct LosslessCut project file import support 