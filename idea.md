# original idea

i have these videos of gameplay
i was thinking, usually when fun things are happening, people are loud
how would we go about figuring out where these points are and turning them into edit markers

# plan

## Feasibility Assessment ✅
- **Core concept is solid** - audio analysis for excitement detection is well-established
- **Technical foundation exists** - FFmpeg, librosa, Web Audio API can handle the heavy lifting
- **Export compatibility** - can generate EDL/XML markers for major video editors

## Phase 1: MVP (Proof of Concept)
1. **Basic volume spike detection**
   - Extract audio track from video files
   - Calculate RMS/peak levels over time windows (1-2 second intervals)
   - Detect spikes above rolling baseline (e.g., 2x average volume)
   - Generate timestamp markers

2. **Simple filtering**
   - Focus on human voice frequency range (85-255 Hz fundamental)
   - Use rolling average to establish per-person baseline
   - Filter out brief spikes (< 1 second) to avoid false positives

## Phase 2: Smart Detection
1. **Multi-signal approach**
   - Rate of volume change (sudden vs gradual increases)
   - Speech pattern analysis (rapid talking detection)
   - Frequency spectrum changes indicating excitement

2. **False positive reduction**
   - Game audio vs voice separation
   - Background noise filtering
   - Context-aware thresholds

## Phase 3: Advanced Features
1. **Machine learning enhancement**
   - Train on manually labeled exciting vs regular moments
   - Learn individual streamer patterns
   - Improve accuracy over time

2. **Additional signals**
   - Mouse movement spike detection
   - Keyboard activity patterns
   - Chat activity correlation (if available)

## Tech Stack Ideas
- **Backend**: goland for processing
- **Audio**: FFmpeg for extraction, librosa/Web Audio for analysis
- **Output**: CSV timestamps (LosslessCut compatible), EDL/XML/CUE optional
- **UI**: Simple web interface for upload/processing

## LosslessCut Integration 🎯
- **Primary output**: CSV format with timestamps (seconds or HH:MM:SS)
- **Format**: `start_time,end_time,label` or `start_time,end_time,description`
- **Import process**: File → Import project → CSV segments
- **Alternative formats**: EDL, YouTube chapters, XML (DaVinci/Final Cut Pro)