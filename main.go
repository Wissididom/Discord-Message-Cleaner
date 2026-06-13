package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	discordAPIURL = "https://discord.com/api/v10"
	userAgent     = "OldMessageDeletor (wissididom.de, 1)"
)

type Message struct {
	ID        string    `json:"id"`
	Author    Author    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

type Author struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type ParsedMessage struct {
	ID         string
	AuthorName string
	Created    string
	Content    string
}

func fetchWithRateLimitHandling(url string, method string, token string, reason *string) ([]byte, error) {
	for {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bot "+token)
		req.Header.Set("User-Agent", userAgent)
		if reason != nil {
			req.Header.Set("X-Audit-Log-Reason", *reason)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfter, err := strconv.Atoi(retryAfterStr)
			if err != nil {
				retryAfter = 0
			}

			if retryAfter > 0 {
				fmt.Printf("%s - Rate limit hit. Retrying after %ds...\n", time.Now().Format("2006-01-02 15:04:05"), retryAfter)
				time.Sleep(time.Duration(retryAfter) * time.Second)
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		return body, nil
	}
}

func getMessages(token string, channelID string, before *string, after *string, limit int) ([]ParsedMessage, error) {
	url := fmt.Sprintf("%s/channels/%s/messages", discordAPIURL, channelID)
	params := []string{}

	if before != nil {
		params = append(params, "before="+*before)
	}
	if after != nil {
		params = append(params, "after="+*after)
	}
	params = append(params, fmt.Sprintf("limit=%d", limit))

	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	body, err := fetchWithRateLimitHandling(url, "GET", token, nil)
	if err != nil {
		return nil, fmt.Errorf("getMessages: %w", err)
	}

	var messages []Message
	err = json.Unmarshal(body, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	var result []ParsedMessage
	for _, msg := range messages {
		result = append(result, ParsedMessage{
			ID:         msg.ID,
			AuthorName: msg.Author.Username,
			Created:    msg.Timestamp.Format(time.RFC3339),
			Content:    msg.Content,
		})
	}

	return result, nil
}

func deleteMessage(token string, channelID string, msgID string, reason *string) (bool, error) {
	url := fmt.Sprintf("%s/channels/%s/messages/%s", discordAPIURL, channelID, msgID)

	if reason == nil {
		defaultReason := "Pepe-Deletor"
		reason = &defaultReason
	}

	_, err := fetchWithRateLimitHandling(url, "DELETE", token, reason)
	if err != nil {
		return false, err
	}

	return true, nil
}

func confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt + " (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load .env file: %v\n", err)
	}

	discordToken := os.Getenv("DISCORD_TOKEN")
	guildID := os.Getenv("SERVER_ID")
	channelID := os.Getenv("CHANNEL_ID")

	if discordToken == "" || guildID == "" || channelID == "" {
		fmt.Fprintf(os.Stderr, "Missing required environment variables: DISCORD_TOKEN, SERVER_ID, CHANNEL_ID\n")
		os.Exit(1)
	}

	var after *string
	if strings.ToLower(os.Getenv("START_WITH_OLDEST")) == "true" {
		afterStr := "1"
		after = &afterStr
	}

	messages, err := getMessages(discordToken, channelID, nil, after, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching messages: %v\n", err)
		os.Exit(1)
	}

	if after != nil && *after == "1" {
		// Reverse to start with oldest
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	for _, message := range messages {
		content := message.Content
		if content == "" {
			content = "N/A"
		}
		timestamp, _ := time.Parse(time.RFC3339, message.Created)
		fmt.Printf("%s - %s: %s\n", timestamp.Format("2006-01-02 15:04:05"), message.AuthorName, content)
	}

	if !confirm("Do you want to delete the above messages?") {
		fmt.Println("Cancelled.")
		return
	}

	for i, message := range messages {
		content := message.Content
		if content == "" {
			content = "N/A"
		}
		timestamp, _ := time.Parse(time.RFC3339, message.Created)
		formattedTime := timestamp.Format("2006-01-02 15:04:05")

		fmt.Printf("[%d/%d] Deleting %s %s - %s: %s\n", i+1, len(messages), message.ID, formattedTime, message.AuthorName, content)

		_, err := deleteMessage(discordToken, channelID, message.ID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting message %s: %v\n", message.ID, err)
			continue
		}

		fmt.Printf("[%d/%d] Deleted %s %s - %s: %s\n", i+1, len(messages), message.ID, formattedTime, message.AuthorName, content)
	}
}
