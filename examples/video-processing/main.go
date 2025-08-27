package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// VideoFile represents a video file to be processed
type VideoFile struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Duration int    `json:"duration"` // in seconds
}

// ProcessingResult represents the result of video processing steps
type ProcessingResult struct {
	Step      string        `json:"step"`
	Success   bool          `json:"success"`
	OutputURL string        `json:"output_url,omitempty"`
	Duration  time.Duration `json:"duration"`
	Message   string        `json:"message"`
}

// PROPOSAL: We need TaskWithContext that receives orchestrator.Context
// This would enable:
// 1. Thread-safe data sharing via ctx.Set/Get
// 2. Orchestration control from within tasks
// 3. Access to orchestration state and configuration
//
// Ideal API would be:
// orchestrator.TaskWithContext(func(ctx orchestrator.Context) (ProcessingResult, error) {
//     video := ctx.Get("video").(VideoFile)  // Access shared data
//     // ... task logic
//     return result, nil
// })
//
// Current limitation: Tasks use closures and can't access orchestrator context

func main() {
	fmt.Println("🎬 Video Streaming Pipeline - Content Processing")
	fmt.Println("Real-world example: YouTube/Netflix video processing")
	fmt.Println()

	// Sample video file
	video := VideoFile{
		ID:       "VID-2025-001",
		Filename: "sample_video.mp4",
		Size:     1024 * 1024 * 500, // 500MB
		Duration: 300,               // 5 minutes
	}

	fmt.Printf("Processing Video: %s\n", video.Filename)
	fmt.Printf("Size: %.1f MB, Duration: %d seconds\n\n", float64(video.Size)/(1024*1024), video.Duration)

	start := time.Now()

	// DEMONSTRATION: Both closure-based and context-aware approaches
	workflow := orchestrator.Setup(
		orchestrator.Sequential(
			// Context-aware task that stores video data for other tasks
			orchestrator.Task(func(ctx orchestrator.Context) (ProcessingResult, error) {
				ctx.Set("video", video) // Store video in orchestrator context
				return ProcessingResult{
					Step:    "context-setup",
					Success: true,
					Message: "Video data stored in orchestrator context",
				}, nil
			}).Named("setup"),

			// Context-aware upload task
			orchestrator.Task(uploadVideoToStorage).Named("upload"),

			// Parallel transcoding - clean one-liner approach
			orchestrator.Concurrent(
				orchestrator.Task(transcodeToHD).Named("hd-transcode"),
				orchestrator.Task(transcodeToSD).Named("sd-transcode"),
				orchestrator.Task(transcodeToMobile).Named("mobile-transcode"),
				orchestrator.Task(generateThumbnails).Named("thumbnails"),
				orchestrator.Task(extractAudioTrack).Named("audio-extraction"),
			).Named("parallel-processing"),

			// Context-aware CDN distribution
			orchestrator.Task(distributeToGlobalCDN).Named("cdn-distribution"),
		).Named("video-pipeline"),
	).OnProgress(func(progress orchestrator.Progress) {
		// Real-time progress updates
		fmt.Printf("📊 Progress: %.1f%% - %s\n", progress.Percentage, progress.Message)
	}).OnStatusChange(func(oldStatus, newStatus orchestrator.Status) {
		fmt.Printf("🔄 Status: %s -> %s\n", oldStatus, newStatus)
	})

	// Execute with progress monitoring
	result, err := workflow.Await()
	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Video processing failed: %v", err)
		return
	}

	fmt.Printf("\n✅ Video processing completed in %v\n\n", duration)
	displayProcessingResults(result, video)
}

// Helper function to get video from context
func getVideoFromContext(ctx orchestrator.Context) (VideoFile, error) {
	videoData := ctx.Get("video")
	if videoData == nil {
		return VideoFile{}, fmt.Errorf("video data not found in context")
	}
	video, ok := videoData.(VideoFile)
	if !ok {
		return VideoFile{}, fmt.Errorf("invalid video data type in context")
	}
	return video, nil
}

// Context-aware video processing functions

func uploadVideoToStorage(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   ☁️  Uploading video to cloud storage...")

	// Simulate upload time based on file size
	uploadTime := time.Duration(video.Size/1024/1024*10) * time.Millisecond // 10ms per MB
	time.Sleep(uploadTime)

	fmt.Println("   ✅ Video uploaded to storage")
	return ProcessingResult{
		Step:      "upload",
		Success:   true,
		OutputURL: fmt.Sprintf("https://storage.example.com/videos/%s", video.ID),
		Duration:  time.Since(start),
		Message:   "Video uploaded to cloud storage",
	}, nil
}

func transcodeToHD(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   🎥 Transcoding to HD (1080p)...")

	// Simulate transcoding time (longer for HD)
	transcodeTime := time.Duration(video.Duration*8) * time.Millisecond // 8ms per second of video
	time.Sleep(transcodeTime)

	fmt.Println("   ✅ HD transcoding completed")
	return ProcessingResult{
		Step:      "hd-transcode",
		Success:   true,
		OutputURL: fmt.Sprintf("https://cdn.example.com/videos/%s_hd.mp4", video.ID),
		Duration:  time.Since(start),
		Message:   "HD (1080p) version created",
	}, nil
}

func transcodeToSD(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   📺 Transcoding to SD (720p)...")

	// Simulate transcoding time
	transcodeTime := time.Duration(video.Duration*5) * time.Millisecond // 5ms per second
	time.Sleep(transcodeTime)

	fmt.Println("   ✅ SD transcoding completed")
	return ProcessingResult{
		Step:      "sd-transcode",
		Success:   true,
		OutputURL: fmt.Sprintf("https://cdn.example.com/videos/%s_sd.mp4", video.ID),
		Duration:  time.Since(start),
		Message:   "SD (720p) version created",
	}, nil
}

func transcodeToMobile(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   📱 Transcoding to mobile (480p)...")

	// Simulate transcoding time (faster for mobile)
	transcodeTime := time.Duration(video.Duration*3) * time.Millisecond // 3ms per second
	time.Sleep(transcodeTime)

	fmt.Println("   ✅ Mobile transcoding completed")
	return ProcessingResult{
		Step:      "mobile-transcode",
		Success:   true,
		OutputURL: fmt.Sprintf("https://cdn.example.com/videos/%s_mobile.mp4", video.ID),
		Duration:  time.Since(start),
		Message:   "Mobile (480p) version created",
	}, nil
}

func generateThumbnails(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   🖼️  Generating thumbnails...")

	// Simulate thumbnail generation
	time.Sleep(200 * time.Millisecond)

	fmt.Println("   ✅ Thumbnails generated")
	return ProcessingResult{
		Step:      "thumbnails",
		Success:   true,
		OutputURL: fmt.Sprintf("https://cdn.example.com/thumbnails/%s/", video.ID),
		Duration:  time.Since(start),
		Message:   "Video thumbnails generated",
	}, nil
}

func extractAudioTrack(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   🎵 Extracting audio track...")

	// Simulate audio extraction
	extractTime := time.Duration(video.Duration*2) * time.Millisecond // 2ms per second
	time.Sleep(extractTime)

	fmt.Println("   ✅ Audio track extracted")
	return ProcessingResult{
		Step:      "audio-extraction",
		Success:   true,
		OutputURL: fmt.Sprintf("https://cdn.example.com/audio/%s.mp3", video.ID),
		Duration:  time.Since(start),
		Message:   "Audio track extracted",
	}, nil
}

func distributeToGlobalCDN(ctx orchestrator.Context) (ProcessingResult, error) {
	video, err := getVideoFromContext(ctx)
	if err != nil {
		return ProcessingResult{}, err
	}
	start := time.Now()
	fmt.Println("   🌍 Distributing to global CDN...")

	// Simulate CDN distribution
	time.Sleep(300 * time.Millisecond)

	fmt.Println("   ✅ Distributed to global CDN")
	return ProcessingResult{
		Step:      "cdn-distribution",
		Success:   true,
		OutputURL: fmt.Sprintf("https://global-cdn.example.com/videos/%s/", video.ID),
		Duration:  time.Since(start),
		Message:   "Video distributed to global CDN",
	}, nil
}

func displayProcessingResults(result *orchestrator.Result, video VideoFile) {
	fmt.Println("📊 Video Processing Results")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Video ID: %s\n", video.ID)
	fmt.Printf("Original File: %s\n\n", video.Filename)

	steps := []string{"upload", "hd-transcode", "sd-transcode", "mobile-transcode",
		"thumbnails", "audio-extraction", "cdn-distribution"}

	totalDuration := time.Duration(0)

	for _, step := range steps {
		if stepResult := result.Get(step); stepResult != nil {
			if procResult, ok := stepResult.(ProcessingResult); ok {
				status := "✅"
				if !procResult.Success {
					status = "❌"
				}
				fmt.Printf("%s %-18s | %8v | %s\n",
					status, procResult.Step, procResult.Duration, procResult.Message)
				if procResult.OutputURL != "" {
					fmt.Printf("   📎 %s\n", procResult.OutputURL)
				}
				totalDuration += procResult.Duration
			}
		}
	}

	fmt.Printf("\n⏱️  Total processing time: %v\n", totalDuration)
	fmt.Println("\n🎉 Video is now available for streaming worldwide!")
}
