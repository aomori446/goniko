package main

import (
	"fmt"
	"goniko/cmd"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func main() {
	cookieCmd := cmd.New("cookie", "Manage cookies")
	rawCookie := cookieCmd.String("raw", "", "raw cookie string", true)
	cookieFile := cookieCmd.String("o", "cookies.txt", "output cookie file path", true)
	cookieCmd.Action = func() {
		_, err := SaveCookie(*rawCookie, ".nicovideo.jp", *cookieFile)
		Must(err)
	}

	rootCmd := cmd.New(os.Args[0], "Download videos from Nikonama")
	url := rootCmd.String("u", "", "watch URL", true)
	cookiePath := rootCmd.String("i", "cookies.txt", "cookie file path", true)
	outputFile := rootCmd.String("o", "", "output MP4/MP3 file path", true)
	audioOnly := rootCmd.Bool("a", false, "download audio(MP3) file only", false)

	rootCmd.Action = func() {
		audioURLs, err := ParseAudioM3U8(*url, *cookiePath)
		Must(err)
		bestAudioURL, err := Best(audioURLs)
		Must(err)

		tempAudio := fmt.Sprintf("%d_audio.mp4", time.Now().UnixNano())
		logger.Info("Downloading audio stream...")
		Must(FormatAndSave(bestAudioURL, *cookiePath, tempAudio))
		defer os.Remove(tempAudio)

		if *audioOnly {
			if filepath.Ext(*outputFile) != ".mp3" {
				Must(fmt.Errorf("now only support .mp3 output, but get %s", *outputFile))
			}
			ffmpegCmd := exec.Command(
				"ffmpeg",
				"-i", tempAudio,
				"-vn",
				"-c:a", "mp3",
				*outputFile,
			)
			ffmpegCmd.Stdout = os.Stdout
			ffmpegCmd.Stderr = os.Stderr
			logger.Info("Converting audio to MP3...", "output", *outputFile)
			Must(ffmpegCmd.Run())
			logger.Info("✅ Audio download complete", "file", *outputFile)
			return
		}

		if filepath.Ext(*outputFile) != ".mp4" {
			Must(fmt.Errorf("now only support .mp4 output, but get %s", *outputFile))
		}

		videoURLs, err := ParseVideoM3U8(*url, *cookiePath)
		Must(err)
		bestVideoURL, err := Best(videoURLs)
		Must(err)
		tempVideo := fmt.Sprintf("%d_video.mp4", time.Now().UnixNano())
		logger.Info("Downloading video stream...")
		Must(FormatAndSave(bestVideoURL, *cookiePath, tempVideo))
		defer os.Remove(tempVideo)

		ffmpegCmd := exec.Command(
			"ffmpeg",
			"-i", tempVideo,
			"-i", tempAudio,
			"-c:v", "copy",
			"-c:a", "copy",
			*outputFile,
		)
		ffmpegCmd.Stdout = os.Stdout
		ffmpegCmd.Stderr = os.Stderr
		logger.Info("Combining audio and video streams...", "output", *outputFile)
		Must(ffmpegCmd.Run())

		logger.Info("✅ Download complete", "file", *outputFile)
	}

	rootCmd.AddSubNode(cookieCmd)
	rootCmd.Parse(os.Args[1:])
}
