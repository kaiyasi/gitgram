# GitHub Activity to Telegram Plan

## 目標

用 Go 做一個服務，接收 GitHub repository 的 Webhook，將 Pull Request、Issue、Commit、Release、CI、Comment 等活動整理成 Telegram 訊息並送到指定 chat。

核心流程：

1. GitHub repo 觸發 Webhook。
2. Go HTTP server 收到事件。
3. 驗證 GitHub Webhook secret 與事件來源。
4. 解析不同事件 payload。
5. 轉成可讀的 Telegram 訊息。
6. 呼叫 Telegram Bot API `sendMessage` 發送。

## MVP 範圍

第一版先支援最常用活動：

- `push`: commits push 到 branch。
- `pull_request`: PR opened, closed, merged, reopened, synchronized。
- `issues`: issue opened, closed, reopened。
- `issue_comment`: issue 或 PR conversation comment。
- `pull_request_review`: PR review submitted。
- `release`: release published。
- `workflow_run`: GitHub Actions workflow completed 且 `conclusion=failure`，只通知失敗的 CI。

MVP 不需要資料庫，但要保留介面，之後可加 SQLite/Postgres/Redis 做去重與追蹤。

## 建議架構

```text
GitHub Webhook
      |
      v
Go HTTP Server
      |
      +-- verify GitHub signature
      +-- parse event payload
      +-- normalize event
      +-- format Telegram message
      +-- send Telegram message
      |
      v
Telegram Chat / Group / Channel
```

建議先做成單一 binary，部署時只需要環境變數即可。

## Go 專案結構

```text
gitgram/
  cmd/gitgram/main.go
  internal/config/config.go
  internal/githubwebhook/handler.go
  internal/githubwebhook/verify.go
  internal/githubwebhook/events.go
  internal/telegram/client.go
  internal/formatter/telegram.go
  internal/router/router.go
  internal/store/store.go
  internal/store/memory.go
  go.mod
  README.md
```

各模組責任：

- `cmd/gitgram/main.go`: 啟動服務、載入設定、註冊 route。
- `internal/config`: 讀取環境變數與設定。
- `internal/githubwebhook`: 驗證與解析 GitHub Webhook。
- `internal/formatter`: 把 GitHub event 轉成 Telegram 文字。
- `internal/telegram`: 封裝 Telegram Bot API。
- `internal/store`: 事件去重與狀態儲存，MVP 可先用 memory。
- `internal/router`: HTTP route 組裝。

## 環境變數

```text
PORT=8080
GITHUB_WEBHOOK_SECRET=your-github-webhook-secret
TELEGRAM_BOT_TOKEN=123456:telegram-bot-token
TELEGRAM_CHAT_ID=-1001234567890
PUBLIC_BASE_URL=https://your-domain.example
ALLOWED_REPOS=owner/repo-a,owner/repo-b
```

說明：

- `GITHUB_WEBHOOK_SECRET`: GitHub Webhook 設定中的 secret，用來驗證 `X-Hub-Signature-256`。
- `TELEGRAM_BOT_TOKEN`: 從 BotFather 建立 bot 後取得。
- `TELEGRAM_CHAT_ID`: 要發送到的 group/channel/user chat id。
- `ALLOWED_REPOS`: 防止任意 repo 打進 webhook。

## GitHub Webhook Endpoint

HTTP endpoint 建議：

```text
POST /webhooks/github
GET  /healthz
```

GitHub Webhook 設定：

- Payload URL: `https://your-domain.example/webhooks/github`
- Content type: `application/json`
- Secret: 與 `GITHUB_WEBHOOK_SECRET` 一致
- Events: 選擇 `Send me everything`，或先勾 MVP 需要的事件

收到 request 後要讀取這些 headers：

- `X-GitHub-Event`: 事件類型，例如 `push`、`pull_request`。
- `X-GitHub-Delivery`: GitHub delivery id，可用於去重。
- `X-Hub-Signature-256`: HMAC SHA-256 簽章。

## Webhook 驗證

驗證方式：

1. 讀取 raw request body。
2. 用 `GITHUB_WEBHOOK_SECRET` 對 body 做 HMAC SHA-256。
3. 格式組成 `sha256=<hex digest>`。
4. 用 constant-time comparison 比對 `X-Hub-Signature-256`。

Go 可用：

```go
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(body)
expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
ok := hmac.Equal([]byte(expected), []byte(signature))
```

也可以使用 `github.com/google/go-github/v66/github` 的 webhook helper 來驗證與解析 payload。

## Telegram 發送

Telegram Bot API endpoint：

```text
POST https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/sendMessage
```

Request body：

```json
{
  "chat_id": "-1001234567890",
  "text": "<b>owner/repo</b> pull request opened\n#12 Add login flow\nhttps://github.com/owner/repo/pull/12",
  "parse_mode": "HTML",
  "disable_web_page_preview": true
}
```

建議使用 `HTML` parse mode，因為 GitHub title 可能包含 Markdown 特殊字元。送出前要 escape 使用者可控文字，例如 issue title、PR title、comment body。

## Event Normalization

不同 GitHub event payload 結構不同，建議先轉成內部統一模型：

```go
type Activity struct {
    DeliveryID string
    Repo       string
    Type       string
    Action     string
    Title      string
    Actor      string
    URL        string
    Branch     string
    CommitSHA  string
    Summary    string
}
```

範例：

- `pull_request/opened` -> `Type=Pull Request`, `Action=opened`
- `issues/closed` -> `Type=Issue`, `Action=closed`
- `push` -> `Type=Push`, `Action=committed`
- `release/published` -> `Type=Release`, `Action=published`

## Message Format

訊息要短、穩定、可掃讀。

Push：

```text
owner/repo push to main
by octocat

abc1234 Fix login redirect
def5678 Add tests

https://github.com/owner/repo/compare/old...new
```

Pull Request：

```text
owner/repo pull request opened
#12 Add login flow
by octocat

https://github.com/owner/repo/pull/12
```

Issue：

```text
owner/repo issue closed
#42 Cannot login with OAuth
by octocat

https://github.com/owner/repo/issues/42
```

Workflow 失敗：

```text
owner/repo workflow completed: failure
CI on main
by octocat

https://github.com/owner/repo/actions/runs/123
```

## 去重與可靠性

GitHub 可能重送 Webhook，所以要用 `X-GitHub-Delivery` 去重。

MVP：

- 用 in-memory map 記錄最近 N 筆 delivery id。
- 服務重啟後可能重送，先接受。

正式版：

- 用 SQLite/Postgres/Redis 記錄 delivery id。
- 記錄 `received_at`、`processed_at`、`status`、`error`。
- Telegram 發送失敗時 retry。

建議狀態：

```text
received -> processing -> sent
received -> processing -> failed -> retrying -> sent
```

## Rate Limit 與 Queue

Telegram 有發送頻率限制，之後應加 queue：

- HTTP handler 只負責驗證、解析、入列，快速回 `200 OK`。
- 背景 worker 從 queue 發送 Telegram。
- 發送失敗做 exponential backoff。

MVP 可同步發送；如果 repo 活動量高，應盡快改成 queue。

## 多 Repo / 多 Chat 支援

第一版可以全部送同一個 `TELEGRAM_CHAT_ID`。

後續可支援設定檔：

```yaml
routes:
  - repo: owner/repo-a
    chat_id: "-100111"
  - repo: owner/repo-b
    chat_id: "-100222"
  - repo: owner/*
    chat_id: "-100333"
```

這樣不同 repo 可以送到不同 Telegram group 或 topic。

## 安全性

必要項目：

- 必須驗證 `X-Hub-Signature-256`。
- 限制 `ALLOWED_REPOS`。
- 不把 secret/token 印到 log。
- Telegram message 要 escape HTML。
- 設定 request body size limit，例如 10 MB。
- 設定 HTTP server read/write timeout。

可選項目：

- 記錄 GitHub delivery id 方便追查。
- 增加 IP allowlist，但不要只依賴 IP。
- 對 `/webhooks/github` 只接受 `POST`。

## 測試策略

單元測試：

- Webhook signature 驗證成功與失敗。
- 不同 event payload 轉成 `Activity`。
- Telegram HTML escape。
- 訊息長度截斷，避免超過 Telegram 限制。

整合測試：

- 用 GitHub Webhook sample payload 測 handler。
- mock Telegram API，確認送出的 JSON 正確。
- 重送相同 `X-GitHub-Delivery` 時不重複發送。

手動測試：

1. 用 BotFather 建 bot。
2. 把 bot 加到 Telegram group。
3. 取得 group chat id。
4. 本機用 `ngrok` 或 `cloudflared tunnel` 暴露 `/webhooks/github`。
5. GitHub repo 新增 Webhook。
6. 建 issue、開 PR、push commit，確認 Telegram 收到訊息。

## 實作里程碑

### Milestone 1: 基本服務

- 建立 Go module。
- 實作 `/healthz`。
- 實作 `/webhooks/github`。
- 讀取環境變數。
- 加 HTTP server timeout。

### Milestone 2: GitHub Webhook

- 驗證 `X-Hub-Signature-256`。
- 讀取 `X-GitHub-Event` 與 `X-GitHub-Delivery`。
- 解析 `push`、`pull_request`、`issues`。
- 加 `ALLOWED_REPOS` 檢查。

### Milestone 3: Telegram

- 實作 Telegram client。
- 支援 `sendMessage`。
- Escape HTML。
- 實作基本 message formatter。

### Milestone 4: 更多事件

- 加 `issue_comment`。
- 加 `pull_request_review`。
- 加 `release`。
- 加 `workflow_run`，但只處理失敗的 workflow run。

### Milestone 5: 可靠性

- 用 delivery id 去重。
- 加 retry。
- 加 queue worker。
- 加 structured logging。

### Milestone 6: 部署

- 寫 Dockerfile。
- 加 graceful shutdown。
- 部署到 Fly.io、Render、Railway、Cloud Run 或 VPS。
- 設定 HTTPS 網域。

## 推薦 Go 套件

- HTTP router: `github.com/go-chi/chi/v5`
- GitHub webhook helper: `github.com/google/go-github/v66/github`
- Config: 先用標準函式庫 `os.Getenv` 即可
- Logging: `log/slog`
- Testing: 標準 `testing` + `httptest`
- Optional queue: 先用 Go channel，正式版再換 Redis 或資料庫

Go 標準函式庫已足夠完成 MVP；第三方套件只在能明顯降低維護成本時再加。

## 第一版主流程偽碼

```go
func GitHubWebhookHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body := readLimitedBody(r.Body)

    if !verifyGitHubSignature(body, r.Header.Get("X-Hub-Signature-256"), secret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    eventType := r.Header.Get("X-GitHub-Event")
    deliveryID := r.Header.Get("X-GitHub-Delivery")

    if store.Seen(deliveryID) {
        w.WriteHeader(http.StatusOK)
        return
    }

    activity, err := ParseGitHubEvent(eventType, body)
    if err != nil {
        http.Error(w, "unsupported event", http.StatusBadRequest)
        return
    }

    if !allowedRepo(activity.Repo) {
        http.Error(w, "repo not allowed", http.StatusForbidden)
        return
    }

    msg := formatter.Telegram(activity)
    err = telegram.SendMessage(ctx, chatID, msg)
    if err != nil {
        http.Error(w, "send failed", http.StatusBadGateway)
        return
    }

    store.MarkSeen(deliveryID)
    w.WriteHeader(http.StatusOK)
}
```

## 需要先決定的問題

實作前建議先確認：

1. 要監控單一 repo，還是多個 repo？
2. 全部活動都送同一個 Telegram chat，還是依 repo 分流？
3. Telegram 目標是 group、channel，還是 topic？
4. 要不要包含 comment body？如果 repo 是 private，comment 內容可能需要更謹慎。
5. 發送失敗時是否需要保證一定補送？

## 建議第一步

先做最小可用版本：

1. `go mod init github.com/<you>/gitgram`
2. 建 `/healthz`
3. 建 `/webhooks/github`
4. 支援 `push`、`pull_request`、`issues`
5. 送到單一 Telegram chat
6. 本機用 tunnel 測 GitHub Webhook

等確認訊息格式與 Telegram group 使用方式符合需求後，再補 queue、資料庫與多 repo routing。
