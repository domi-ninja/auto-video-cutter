package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-audio/wav"
)

type ExcitementMarker struct {
	StartTime float64
	EndTime   float64
	Label     string
	Score     float64
}

type CutSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Name  string  `json:"name"`
}

type LosslessCutProject struct {
	Version       int          `json:"version"`
	MediaFileName string       `json:"mediaFileName"`
	CutSegments   []CutSegment `json:"cutSegments"`
}

type AudioAnalyzer struct {
	WindowSize     int     // Window size in samples
	ThresholdRatio float64 // Multiplier for baseline volume
	MinDuration    float64 // Minimum duration for valid excitement (seconds)
	SampleRate     int
}

func main() {
	var (
		inputFile   = flag.String("input", "", "Input video file path")
		outputFile  = flag.String("output", "", "Output LosslessCut project file path (default: input_name.proj.llc)")
		threshold   = flag.Float64("threshold", 2.0, "Volume spike threshold multiplier")
		minDuration = flag.Float64("min-duration", 1.0, "Minimum excitement duration in seconds")
		windowMs    = flag.Int("window", 1000, "Analysis window size in milliseconds")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: video-cutter -input <video_file> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *outputFile == "" {
		ext := filepath.Ext(*inputFile)
		base := strings.TrimSuffix(filepath.Base(*inputFile), ext)
		*outputFile = base + ".proj.llc"
	}

	if *verbose {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(io.Discard)
	}

	fmt.Printf("Processing video: %s\n", *inputFile)
	fmt.Printf("Output file: %s\n", *outputFile)

	// Extract audio from video
	audioFile, err := extractAudio(*inputFile)
	if err != nil {
		log.Fatalf("Failed to extract audio: %v", err)
	}
	defer os.Remove(audioFile) // Clean up temp audio file

	// Analyze audio for excitement markers
	analyzer := &AudioAnalyzer{
		WindowSize:     *windowMs * 44100 / 1000, // Convert ms to samples (44.1kHz)
		ThresholdRatio: *threshold,
		MinDuration:    *minDuration,
		SampleRate:     44100,
	}

	markers, err := analyzer.AnalyzeAudio(audioFile)
	if err != nil {
		log.Fatalf("Failed to analyze audio: %v", err)
	}

	cleanedUpMarkers := mergeOverlappingMarkers(markers)

	// Export markers to LosslessCut JSON format
	err = exportToLosslessCut(cleanedUpMarkers, *outputFile, filepath.Base(*inputFile))
	if err != nil {
		log.Fatalf("Failed to export markers: %v", err)
	}

	fmt.Printf("Found %d excitement markers\n", len(markers))
	fmt.Printf("Cleaned up %d excitement markers\n", len(cleanedUpMarkers))
	fmt.Printf("Markers exported to: %s\n", *outputFile)
	fmt.Println("Import this file into LosslessCut: File → Open → Select the .proj.llc file")
}

func extractAudio(videoFile string) (string, error) {
	tempDir := os.TempDir()
	audioFile := filepath.Join(tempDir, "temp_audio.wav")

	// Use FFmpeg to extract audio with optimizations for speed
	cmd := exec.Command("ffmpeg",
		"-i", videoFile,
		"-vn",                  // No video
		"-acodec", "pcm_s16le", // 16-bit PCM
		"-ar", "44100", // Use consistent 44.1kHz sample rate
		"-ac", "1", // Mono
		"-threads", "0", // Use all available CPU cores
		"-preset", "ultrafast", // Fastest encoding preset
		"-y", // Overwrite output file
		audioFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Audio extracted to: %s", audioFile)
	return audioFile, nil
}

func (a *AudioAnalyzer) AnalyzeAudio(audioFile string) ([]ExcitementMarker, error) {
	// Open WAV file
	file, err := os.Open(audioFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// Read all audio data
	audioData, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	// Convert integer samples to float64
	intSamples := audioData.Data
	samples := make([]float64, len(intSamples))
	maxValue := math.Pow(2, float64(decoder.BitDepth-1)) // 2^(bitDepth-1) for signed integers

	for i, sample := range intSamples {
		samples[i] = float64(sample) / maxValue
	}

	sampleRate := float64(decoder.SampleRate)
	a.SampleRate = int(sampleRate)
	// No need to adjust WindowSize since we're using consistent 44.1kHz

	log.Printf("Audio info: %d samples, %.1f Hz, %.2f seconds",
		len(samples), sampleRate, float64(len(samples))/sampleRate)

	return a.detectExcitementMarkers(samples, sampleRate), nil
}

func (a *AudioAnalyzer) detectExcitementMarkers(samples []float64, sampleRate float64) []ExcitementMarker {
	if len(samples) == 0 {
		return []ExcitementMarker{}
	}

	// Calculate RMS (Root Mean Square) values for sliding windows
	windowSamples := a.WindowSize
	if windowSamples <= 0 {
		windowSamples = int(sampleRate) // Default to 1 second
	}

	numWindows := len(samples) / windowSamples
	if numWindows == 0 {
		return []ExcitementMarker{}
	}

	rmsValues := make([]float64, numWindows)

	// Calculate RMS for each window
	for i := 0; i < numWindows; i++ {
		start := i * windowSamples
		end := start + windowSamples
		if end > len(samples) {
			end = len(samples)
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += samples[j] * samples[j]
		}
		rmsValues[i] = math.Sqrt(sum / float64(end-start))
	}

	// Calculate baseline (average RMS)
	baseline := 0.0
	for _, rms := range rmsValues {
		baseline += rms
	}
	baseline /= float64(len(rmsValues))

	log.Printf("Baseline RMS: %.6f", baseline)

	threshold := baseline * a.ThresholdRatio
	log.Printf("Threshold: %.6f (%.1fx baseline)", threshold, a.ThresholdRatio)

	// Find excitement periods
	var markers []ExcitementMarker
	var excitementStart int
	var inExcitement bool

	for i, rms := range rmsValues {
		if rms > threshold {
			if !inExcitement {
				// Start of excitement period
				excitementStart = i
				inExcitement = true
				log.Printf("Excitement start at window %d (%.2fs), RMS: %.6f", i, float64(i*windowSamples)/sampleRate, rms)
			}
		} else {
			if inExcitement {
				// End of excitement period
				windowDiff := i - excitementStart
				duration := float64(windowDiff) * float64(windowSamples) / sampleRate
				log.Printf("Excitement end at window %d (%.2fs), excitementStart: %d, i: %d, windowDiff: %d, windowSamples: %d, sampleRate: %.0f, duration: %.2fs, min required: %.2fs", i, float64(i*windowSamples)/sampleRate, excitementStart, i, windowDiff, windowSamples, sampleRate, duration, a.MinDuration)
				if duration >= a.MinDuration {
					startTime := float64(excitementStart*windowSamples) / sampleRate
					endTime := float64(i*windowSamples) / sampleRate

					// Calculate average multiplier for this segment
					avgMultiplier := 0.0
					count := 0
					for j := excitementStart; j < i; j++ {
						avgMultiplier += rmsValues[j] / baseline
						count++
					}
					if count > 0 {
						avgMultiplier /= float64(count)
					}

					marker := ExcitementMarker{
						StartTime: startTime,
						EndTime:   endTime,
						Label:     fmt.Sprintf("Excitement (%.1fx)", avgMultiplier),
						Score:     avgMultiplier,
					}
					markers = append(markers, marker)
					log.Printf("Added marker: %.2fs-%.2fs (%.1fx)", startTime, endTime, avgMultiplier)
				} else {
					log.Printf("Skipping short excitement period: %.2fs < %.2fs", duration, a.MinDuration)
				}
				inExcitement = false
			}
		}
	}

	// Handle case where excitement period extends to end of audio
	if inExcitement {
		windowDiff := len(rmsValues) - excitementStart
		duration := float64(windowDiff) * float64(windowSamples) / sampleRate
		if duration >= a.MinDuration {
			startTime := float64(excitementStart*windowSamples) / sampleRate
			endTime := float64(len(samples)) / sampleRate

			// Calculate average multiplier for this segment
			avgMultiplier := 0.0
			count := 0
			for j := excitementStart; j < len(rmsValues); j++ {
				avgMultiplier += rmsValues[j] / baseline
				count++
			}
			if count > 0 {
				avgMultiplier /= float64(count)
			}

			marker := ExcitementMarker{
				StartTime: startTime,
				EndTime:   endTime,
				Label:     fmt.Sprintf("Excitement (%.1fx)", avgMultiplier),
				Score:     avgMultiplier,
			}
			markers = append(markers, marker)
			log.Printf("Added final marker: %.2fs-%.2fs (%.1fx)", startTime, endTime, avgMultiplier)
		}
	}

	return markers
}

func exportToLosslessCut(markers []ExcitementMarker, filename string, mediaFileName string) error {
	project := LosslessCutProject{
		Version:       1,
		MediaFileName: mediaFileName,
		CutSegments:   make([]CutSegment, len(markers)),
	}

	for i, marker := range markers {
		start := marker.StartTime
		end := marker.EndTime

		project.CutSegments[i] = CutSegment{
			Start: start,
			End:   end,
			Name:  marker.Label,
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(project)
	if err != nil {
		return err
	}

	return nil
}

func mergeOverlappingMarkers(markers []ExcitementMarker) []ExcitementMarker {
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].StartTime < markers[j].StartTime
	})

	mergedMarkers := []ExcitementMarker{}
	for _, marker := range markers {
		if len(mergedMarkers) == 0 || mergedMarkers[len(mergedMarkers)-1].EndTime < marker.StartTime {
			mergedMarkers = append(mergedMarkers, marker)
		} else {
			mergedMarkers[len(mergedMarkers)-1].EndTime = math.Max(mergedMarkers[len(mergedMarkers)-1].EndTime, marker.EndTime)
		}
	}
	return mergedMarkers
}
