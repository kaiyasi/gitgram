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

## 訊息格式

服務會使用 Telegram `HTML` parse mode 發送訊息。Repo 名稱會是粗體，branch、actor、commit SHA 會是 monospace，GitHub URL 會顯示成 `Open on GitHub` link。

Push：

```text
owner/repo push to main
by octocat

abc1234 Fix login redirect
def5678 Add tests

Open on GitHub
```

如果一次 push 超過 5 個 commit，只會列前 5 個，最後補一行：

```text
... and 3 more commits
```

Pull Request：

```text
owner/repo pull request opened
#12 Add login flow
by octocat
feature/login -> main
Open on GitHub
```

Issue：

```text
owner/repo issue closed
#42 Cannot login with OAuth
by octocat
Open on GitHub
```

Issue 或 PR comment：

```text
owner/repo pull request comment created
#12 Add login flow
by octocat

Can you add one more test for the redirect case?

Open on GitHub
```

PR comment 會先解析常見 GitHub Markdown，再轉成 Telegram 支援的 HTML。支援範圍包含 bold、italic、strikethrough、inline code、code block、link、blockquote、list、task list。Table 會攤平成文字，raw HTML 會被 escape，unsafe link 會只保留文字。

PR comment body 超過 300 個字元時會被省略，只有 PR comment 會套用這個規則：

```text
owner/repo pull request comment created
#12 Add login flow
by octocat

Very long PR comment body...
Comment truncated. Open on GitHub for full text.

Open on GitHub
```

Pull Request review：

```text
owner/repo pull request review changes requested
#12 Add login flow
by octocat

Please handle the empty token case before merge.

Open on GitHub
```

Release：

```text
owner/repo release published
v1.0.0
by octocat
Open on GitHub
```

GitHub Actions workflow failure：

```text
owner/repo workflow failed
CI on main
by octocat
Open on GitHub
```

`workflow_run` 只會在 `action=completed` 且 `conclusion=failure` 時發送。成功、取消、略過的 workflow run 會被忽略。

## 測試

```sh
go test ./...
```
