package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	playerFooterSelector = `#akashic-gameview`
	playButtonSelector   = `#root button[aria-label="再生 (Space)"]`
)

func ParseRootM3U8URL(rawURL string, cookieFileName string) (rootM3U8URL string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	urlCh := make(chan string, 1)
	var once sync.Once
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if req, ok := ev.(*network.EventRequestWillBeSent); ok {
			if req.Type == network.ResourceTypeXHR && strings.Contains(req.Request.URL, "/playlists/variants/") {
				once.Do(func() { urlCh <- req.Request.URL })
			}
		}
	})

	tasks := chromedp.Tasks{
		network.Enable(),
	}
	cs, err := ReadCookie(cookieFileName)
	if err != nil {
		return "", err
	}
	if len(cs) > 0 {
		tasks = append(tasks, network.SetCookies(cs))
	}
	tasks = append(tasks,
		chromedp.Navigate(rawURL),
		chromedp.WaitVisible(playerFooterSelector),
		chromedp.WaitEnabled(playButtonSelector),
		chromedp.Click(playButtonSelector),
	)

	if err := chromedp.Run(ctx, tasks); err != nil {
		return "", fmt.Errorf("chromedp run error: %w", err)
	}

	select {
	case rootM3U8URL = <-urlCh:
		if rootM3U8URL == "" {
			return "", fmt.Errorf("root M3U8 URL is empty")
		}
		return rootM3U8URL, nil
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for root M3U8: %w", ctx.Err())
	}
}

func ParseRootM3U8(rootM3U8URL string) (audioM3U8URLs []string, videoM3U8URLs []string, err error) {
	resp, err := http.Get(rootM3U8URL)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTPリクエスト失敗: %w", err)
	}
	defer resp.Body.Close()

	audioM3U8URLs = make([]string, 0, 15)
	videoM3U8URLs = make([]string, 0, 15)

	scanner := bufio.NewScanner(resp.Body)
	urlRegex := regexp.MustCompile(`https://[^"]*`)

	for scanner.Scan() {
		line := scanner.Text()
		foundURL := urlRegex.FindString(line)
		if foundURL != "" {
			if strings.Contains(foundURL, "main-audio") {
				audioM3U8URLs = append(audioM3U8URLs, foundURL)
			} else if strings.Contains(foundURL, "main-video") {
				videoM3U8URLs = append(videoM3U8URLs, foundURL)
			} else {
				fmt.Printf("無効なURLをスキップ: %q\n", foundURL)
				continue
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("読み込み中にエラー発生: %w", err)
	}

	if len(audioM3U8URLs) == 0 || len(videoM3U8URLs) == 0 {
		return nil, nil, fmt.Errorf("有効な audio または video URL が見つかりませんでした")
	}

	return audioM3U8URLs, videoM3U8URLs, nil
}

func ParseAudioM3U8(rawURL string, cookieFileName string) (audioM3U8URLs []string, err error) {
	root, err := ParseRootM3U8URL(rawURL, cookieFileName)
	if err != nil {
		return nil, err
	}
	audioM3U8URLs, _, err = ParseRootM3U8(root)
	return
}

func ParseVideoM3U8(rawURL string, cookieFileName string) (audioM3U8URLs []string, err error) {
	root, err := ParseRootM3U8URL(rawURL, cookieFileName)
	if err != nil {
		return nil, err
	}
	_, audioM3U8URLs, err = ParseRootM3U8(root)
	return
}

func FormatAndSave(m3u8URL string, cookieFileName, dstFileName string) error {
	tempF, err := os.CreateTemp("", "*.m3u8")
	if err != nil {
		return fmt.Errorf("一時ファイル作成失敗: %w", err)
	}
	defer os.Remove(tempF.Name())
	defer tempF.Close()

	resp, err := http.Get(m3u8URL)
	if err != nil {
		return fmt.Errorf("m3u8 取得失敗: %w", err)
	}
	defer resp.Body.Close()

	lineNum := 1
	formatOn := false
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		txt := scanner.Text()

		// 6行目に "/init?q=" が含まれる場合のみ formatOn を true
		if lineNum == 6 && strings.Contains(txt, "/init?q=") {
			formatOn = true
		}

		// 10行目までスキップ
		if lineNum <= 10 && formatOn {
			lineNum++
			continue
		}

		_, err = tempF.WriteString(txt + "\n")
		if err != nil {
			return fmt.Errorf("一時ファイル書き込み失敗: %w", err)
		}
		lineNum++
	}

	if err = scanner.Err(); err != nil {
		return fmt.Errorf("m3u8 読み込み中にエラー発生: %w", err)
	}

	cmd := exec.Command(
		"yt-dlp",
		"--cookies", cookieFileName,
		"-o", dstFileName,
		"--enable-file-urls",
		fmt.Sprintf("file://%s", tempF.Name()),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func Best(us []string) (string, error) {
	qualityMap := map[string]int{
		"96Kbps":  1,
		"192Kbps": 2,
		"384Kbps": 3,
		"1Mbps":   4,
		"2Mbps":   5,
		"3Mbps":   6,
	}

	var bestURL string
	var bestQuality int

	for _, u := range us {
		for quality, rank := range qualityMap {
			if strings.Contains(u, quality) && rank > bestQuality {
				bestQuality = rank
				bestURL = u
			}
		}
	}

	if bestURL == "" {
		return "", errors.New("no available url")
	}

	return bestURL, nil
}
