package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"
)

// SlackPayload はSlackに送るJSONの構造体
type SlackPayload struct {
	Text string `json:"text"`
}

func sendToSlack(webhookURL string, message string) error {
	payload := SlackPayload{Text: message}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slackへの送信に失敗しました: %s", resp.Status)
	}
	return nil
}

func main() {
	_ = godotenv.Load() // ローカル開発用

	// 1. 判定基準となる時間を設定 (24時間前)
	// 本番では前回の実行時間をファイルから読み込むのが理想ですが、
	// 毎日8時実行なら「現在から24時間以内」というロジックがシンプルです。
	now := time.Now()
	threshold := now.Add(-24 * time.Hour)

	feeds := []string{
		"https://thehackernews.com/feeds/posts/default",
		"https://www.bleepingcomputer.com/feed/",
	}

	ctx := context.Background()
	client, _ := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	defer client.Close()
	model := client.GenerativeModel("gemini-3-flash-preview")

	fp := gofeed.NewParser()

	for _, url := range feeds {
		feed, err := fp.ParseURL(url)
		if err != nil {
			log.Printf("Feed取得失敗: %v", err)
			continue
		}

		for _, item := range feed.Items {
			// 記事の公開日時が24時間以内か判定
			if item.PublishedParsed != nil && item.PublishedParsed.After(threshold) {

				// 2. Geminiで翻訳
				prompt := fmt.Sprintf(
					"以下のセキュリティ記事のタイトルと概要を日本語に翻訳してください。出力は『タイトル: 翻訳結果\n概要: 翻訳結果』の形式にしてください。\n\nTitle: %s\nDescription: %s",
					item.Title, item.Description,
				)

				resp, err := model.GenerateContent(ctx, genai.Text(prompt))
				if err != nil {
					log.Printf("翻訳失敗: %v", err)
					continue
				}

				translatedText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

				// 3. Slack投稿メッセージ作成
				message := fmt.Sprintf(
					"🛡 *Source: %s*\n%s\n🔗 <%s|記事を読む>",
					feed.Title,
					translatedText,
					item.Link,
				)

				// // ここでSlack送信関数を呼ぶ (前述の http.Post ロジック)
				// fmt.Println("--- Sending to Slack ---")
				// fmt.Println(message)
				webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
				if webhookURL == "" {
					fmt.Println("SLACK_WEBHOOK_URL が設定されていません。コンソール出力のみ行います。")
					return
				}
				err = sendToSlack(webhookURL, message)
				if err != nil {
					fmt.Println("Slack送信エラー:", err)
				} else {
					fmt.Println("Slackにニュースを投稿しました！")
				}

			}
		}
	}
}
