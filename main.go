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
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"
)

// 翻訳対象の記事データを保持する構造体
type TargetArticle struct {
	Source string
	Title  string
	Desc   string
	Link   string
}

// Slack Webhook用の構造体
type SlackPayload struct {
	Text        string `json:"text"`
	UnfurlLinks bool   `json:"unfurl_links"`
	UnfurlMedia bool   `json:"unfurl_media"`
}

func main() {

	// 1. 24時間以内の記事をフィルタリングするための基準時間
	threshold := time.Now().Add(-24 * time.Hour)

	feeds := []string{
		"https://thehackernews.com/feeds/posts/default",
		"https://www.bleepingcomputer.com/feed/",
	}

	var targetArticles []TargetArticle
	fp := gofeed.NewParser()

	// 2. 各フィードから対象記事を抽出
	for _, url := range feeds {
		feed, err := fp.ParseURL(url)
		if err != nil {
			log.Printf("フィード取得失敗 [%s]: %v", url, err)
			continue
		}

		for _, item := range feed.Items {
			pubDate := item.PublishedParsed
			if pubDate == nil {
				pubDate = item.UpdatedParsed
			}

			if pubDate != nil && pubDate.After(threshold) {
				targetArticles = append(targetArticles, TargetArticle{
					Source: feed.Title,
					Title:  item.Title,
					Desc:   item.Description,
					Link:   item.Link,
				})
			}
		}
	}

	if len(targetArticles) == 0 {
		fmt.Println("新着記事はありませんでした。")
		return
	}

	// 3. Gemini API で一括翻訳 (Rate Limit対策)
	translatedContent, err := translateArticlesBulk(targetArticles)
	if err != nil {
		log.Fatalf("翻訳エラー: %v", err)
	}

	// 4. Slackへ送信
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL != "" {
		err := sendToSlack(webhookURL, translatedContent)
		if err != nil {
			log.Fatalf("Slack送信エラー: %v", err)
		}
		fmt.Println("Slackに投稿が完了しました！")
	} else {
		fmt.Println("【デバッグ出力】\n", translatedContent)
	}
}

func translateArticlesBulk(articles []TargetArticle) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3-flash-preview")

	// プロンプトの組み立て
	var bulkText string
	for i, a := range articles {
		bulkText += fmt.Sprintf("[%d] Source: %s\nTitle: %s\nContent: %s\nLink: %s\n\n", i+1, a.Source, a.Title, a.Desc, a.Link)
	}

	prompt := "以下のセキュリティニュースを日本語に翻訳してください。各記事の最後に必ず元のLinkを添えてください。Slackで読みやすいように、タイトルは太字(*タイトル*)にしてください。\n\n" + bulkText

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("📢 *本日の海外セキュリティニュース (%s)*\n\n%v",
		time.Now().AddDate(0, 0, 1).Format("2006/01/02"),
		resp.Candidates[0].Content.Parts[0]), nil
}

func sendToSlack(webhookURL string, message string) error {
	payload := SlackPayload{
		Text:        message,
		UnfurlLinks: false,
		UnfurlMedia: false,
	}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack status error: %s", resp.Status)
	}
	return nil
}
