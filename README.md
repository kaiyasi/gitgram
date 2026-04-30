# gitgram

GitHub Webhook to Telegram notification service written in Go.

## 功能

- 接收 GitHub repository Webhook。
- 驗證 `X-Hub-Signature-256`。
- 支援 `push`、`pull_request`、`issues`、`issue_comment`、`pull_request_review`、`release`。
- 支援 `workflow_run`，但只通知 `conclusion=failure` 的 GitHub Actions 失敗。
- 使用 Telegram Bot API `sendMessage` 發送 HTML 格式訊息。
- 使用 `X-GitHub-Delivery` 做 in-memory 去重。
- 透過 `ALLOWED_REPOS` 限制允許的 repository。

## 環境變數

可以先從範例建立本機設定：

```sh
cp .env.example .env
```

目前服務會讀取 process environment，不會自動載入 `.env` 檔。若用 `.env`，請透過 shell、部署平台或 dotenv 工具載入。

```text
PORT=8080
GITHUB_WEBHOOK_SECRET=your-github-webhook-secret
TELEGRAM_BOT_TOKEN=123456:telegram-bot-token
TELEGRAM_CHAT_ID=-1001234567890
ALLOWED_REPOS=owner/repo-a,owner/repo-b
```

可選：

```text
MAX_BODY_BYTES=10485760
DELIVERY_CACHE_SIZE=10000
PUBLIC_BASE_URL=https://your-domain.example
```

`ALLOWED_REPOS` 支援：

- `owner/repo`
- `owner/*`
- `*`

## 執行

```sh
go run ./cmd/gitgram
```

## Docker

Build image：

```sh
docker build -t gitgram .
```

Run container：

```sh
docker run --rm -p 8080:8080 --env-file .env gitgram
```

健康檢查：

```text
GET /healthz
```

GitHub Webhook：

```text
POST /webhooks/github
```

## GitHub Webhook 設定

- Payload URL: `https://your-domain.example/webhooks/github`
- Content type: `application/json`
- Secret: 和 `GITHUB_WEBHOOK_SECRET` 相同
- Events: 選擇需要的事件，或先用 `Send me everything`

## Telegram 設定

1. 用 BotFather 建立 bot，取得 `TELEGRAM_BOT_TOKEN`。
2. 把 bot 加入 Telegram group 或 channel。
3. 取得目標 chat id，設定到 `TELEGRAM_CHAT_ID`。

## 測試

```sh
go test ./...
```
