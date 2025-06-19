package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-audio/wav"
)

type ExcitementMarker struct {
	StartTime float64
	EndTime   float64
	Label     string
	Score     float64
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
		outputFile  = flag.String("output", "", "Output CSV file path (default: input_name_markers.csv)")
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
		*outputFile = base + "_markers.csv"
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
		WindowSize:     *windowMs * 44100 / 1000, // Convert ms to samples (assuming 44.1kHz)
		ThresholdRatio: *threshold,
		MinDuration:    *minDuration,
		SampleRate:     44100,
	}

	markers, err := analyzer.AnalyzeAudio(audioFile)
	if err != nil {
		log.Fatalf("Failed to analyze audio: %v", err)
	}

	// Export markers to CSV
	err = exportToCSV(markers, *outputFile)
	if err != nil {
		log.Fatalf("Failed to export markers: %v", err)
	}

	fmt.Printf("Found %d excitement markers\n", len(markers))
	fmt.Printf("Markers exported to: %s\n", *outputFile)
	fmt.Println("Import this CSV file into LosslessCut: File → Import project → CSV segments")
}

func extractAudio(videoFile string) (string, error) {
	tempDir := os.TempDir()
	audioFile := filepath.Join(tempDir, "temp_audio.wav")

	// Use FFmpeg to extract audio as 16-bit 44.1kHz WAV
	cmd := exec.Command("ffmpeg",
		"-i", videoFile,
		"-vn",                  // No video
		"-acodec", "pcm_s16le", // 16-bit PCM
		"-ar", "44100", // 44.1kHz sample rate
		"-ac", "1", // Mono
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
	a.WindowSize = int(float64(a.WindowSize) * sampleRate / 44100) // Adjust for actual sample rate

	log.Printf("Audio info: %d samples, %.1f Hz, %.2f seconds",
		len(samples), sampleRate, float64(len(samples))/sampleRate)

	return a.detectExcitementMarkers(samples, sampleRate), nil
}

func (a *AudioAnalyzer) detectExcitementMarkers(samples []float64, sampleRate float64) []ExcitementMarker {
	windowSize := a.WindowSize
	if windowSize > len(samples) {
		windowSize = len(samples)
	}

	var markers []ExcitementMarker
	var volumes []float64

	// Calculate RMS volume for each window
	for i := 0; i < len(samples)-windowSize; i += windowSize / 2 { // 50% overlap
		rms := calculateRMS(samples[i : i+windowSize])
		volumes = append(volumes, rms)
	}

	if len(volumes) < 10 {
		log.Printf("Warning: Not enough audio data for analysis")
		return markers
	}

	// Calculate baseline (median of all volumes)
	baseline := calculateMedian(volumes)
	threshold := baseline * a.ThresholdRatio

	log.Printf("Baseline volume: %.6f, Threshold: %.6f", baseline, threshold)

	// Find excitement periods
	inExcitement := false
	startTime := 0.0
	windowDuration := float64(windowSize/2) / sampleRate // Time per window step

	for i, volume := range volumes {
		currentTime := float64(i) * windowDuration

		if !inExcitement && volume > threshold {
			// Start of excitement
			inExcitement = true
			startTime = currentTime
			log.Printf("Excitement start at %.2fs (volume: %.6f)", startTime, volume)
		} else if inExcitement && volume <= threshold {
			// End of excitement
			duration := currentTime - startTime
			if duration >= a.MinDuration {
				score := volume / baseline
				markers = append(markers, ExcitementMarker{
					StartTime: startTime,
					EndTime:   currentTime,
					Label:     fmt.Sprintf("Excitement (%.1fx)", score),
					Score:     score,
				})
				log.Printf("Excitement end at %.2fs (duration: %.2fs, score: %.1fx)",
					currentTime, duration, score)
			} else {
				log.Printf("Excitement too short: %.2fs", duration)
			}
			inExcitement = false
		}
	}

	// Handle case where excitement continues to end of audio
	if inExcitement {
		endTime := float64(len(samples)) / sampleRate
		duration := endTime - startTime
		if duration >= a.MinDuration {
			markers = append(markers, ExcitementMarker{
				StartTime: startTime,
				EndTime:   endTime,
				Label:     "Excitement (end)",
				Score:     volumes[len(volumes)-1] / baseline,
			})
		}
	}

	return markers
}

func calculateRMS(samples []float64) float64 {
	var sum float64
	for _, sample := range samples {
		sum += sample * sample
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Make a copy and sort it
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if len(sorted)%2 == 0 {
		return (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	return sorted[len(sorted)/2]
}

func exportToCSV(markers []ExcitementMarker, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"start_time", "end_time", "label"})
	if err != nil {
		return err
	}

	// Write markers
	for _, marker := range markers {
		record := []string{
			formatTime(marker.StartTime),
			formatTime(marker.EndTime),
			marker.Label,
		}
		err = writer.Write(record)
		if err != nil {
			return err
		}
	}

	return nil
}

func formatTime(seconds float64) string {
	return strconv.FormatFloat(seconds, 'f', 2, 64)
}
