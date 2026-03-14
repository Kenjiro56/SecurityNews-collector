package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"
)

func main() {
	// 1. RSSフィードの取得 (例: BleepingComputer)
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://www.bleepingcomputer.com/feed/")

	// 最新の1件だけ処理（テスト用）
	item := feed.Items[0]

	// 2. Gemini APIで要約・翻訳
	ctx := context.Background()
	client, _ := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	model := client.GenerativeModel("gemini-1.5-flash")

	prompt := fmt.Sprintf("以下のセキュリティニュースを日本語に翻訳してください:\nTitle: %s\nContent: %s", item.Title, item.Description)
	resp, _ := model.GenerateContent(ctx, genai.Text(prompt))

	// 3. Slackへ送信 (ここは標準のhttp.PostでOK)
	summary := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	fmt.Printf("【翻訳結果】\n%s\n", summary)

	// TODO: Slack通知関数の呼び出し
}
