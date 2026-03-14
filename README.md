# Security News Bot

海外のセキュリティニュースを収集し、Gemini APIで翻訳してSlackへ通知するボットです。

## 概要
* **RSS収集**: The Hacker News, BleepingComputer から24時間以内の新着記事を取得。
* **一括翻訳**: Gemini APIを用いて、タイトルと概要を日本語へ一括翻訳。
* **Slack通知**: Webhookを利用し、リンクプレビュー（OGP）を無効化した状態で通知。
* **自動化**: GitHub Actionsで毎日朝8時（日本時間）に定期実行。

## 使用言語・ツール
* **Language**: Go (Golang)
* **LLM**: gemini-3-flash-preview (Google AI SDK)
* **CI/CD**: GitHub Actions