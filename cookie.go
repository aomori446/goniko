package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
)

func SaveCookie(rawStr, domain, fileName string, filter ...string) ([]*network.CookieParam, error) {
	rawStr = strings.TrimSpace(rawStr)
	if strings.HasPrefix(rawStr, "Cookie: ") {
		rawStr = strings.TrimPrefix(rawStr, "Cookie: ")
	}

	var cookies []*network.CookieParam
	for _, pair := range strings.Split(rawStr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			fmt.Printf("Ignore invalid cookie: %q\n", pair)
			continue
		}

		name := parts[0]
		value := parts[1]

		if len(filter) > 0 && !slices.Contains(filter, name) {
			continue
		}

		cookies = append(cookies, &network.CookieParam{
			Name:     name,
			Value:    value,
			Domain:   domain,
			Path:     "/",
			HTTPOnly: false,
			Secure:   false,
		})
	}

	if len(cookies) == 0 {
		return nil, errors.New("no valid cookies found")
	}

	if fileName == "" {
		return cookies, nil
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if _, err = f.WriteString("# Netscape HTTP Cookie File\n\n"); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	expirationTimestamp := time.Now().Add(10 * 365 * 24 * time.Hour).Unix()

	for _, cookie := range cookies {
		line := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			cookie.Domain,
			"TRUE",
			"/",
			"FALSE",
			expirationTimestamp,
			cookie.Name,
			cookie.Value,
		)
		if _, err = f.WriteString(line); err != nil {
			return nil, fmt.Errorf("failed to write cookie %q: %w", cookie.Name, err)
		}
	}

	return cookies, nil
}

func ReadCookie(fileName string) ([]*network.CookieParam, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	cookies := make([]*network.CookieParam, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 7 {
			fmt.Printf("ignore invalid line: %q\n", line)
			continue
		}

		domain := parts[0]
		// includeSubdomains := parts[1] // not used here
		path := parts[2]
		secure := parts[3] == "TRUE"
		name := parts[5]
		value := parts[6]

		cookies = append(cookies, &network.CookieParam{
			Name:     name,
			Value:    value,
			Domain:   domain,
			Path:     path,
			Secure:   secure,
			HTTPOnly: false,
		})
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	if len(cookies) == 0 {
		return nil, errors.New("no valid cookies found")
	}

	return cookies, nil
}
