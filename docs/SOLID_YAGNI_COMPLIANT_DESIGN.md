# Hokku - MVP優先・段階的SOLID設計仕様書

## 概要
MVPアプローチを採用し、SOLID原則とYAGNI原則を段階的に適用するHokkuの設計です。

## 設計哲学：動く→改善→抽象化

### ❌ 従来の間違ったアプローチ
```
1. インターフェース定義から開始
2. 抽象化層の構築  
3. 依存性注入の実装
4. やっとHTTPハンドラ実装
5. ようやく動作確認
```

### ✅ MVPアプローチ（段階的SOLID適用）
```
Phase 0: 動作する最小コード（YAGNI重視）
Phase 1: 基本品質向上（SRP部分適用）
Phase 2: 必要な抽象化（OCP適用）
Phase 3: 完全なSOLID適用（LSP, ISP, DIP）
```

## 段階的SOLID適用戦略

### Phase 0: 動作最優先（YAGNI徹底）
**目標**: とにかく動くものを作る（30分）
**SOLID適用度**: 0% - 動作が最優先

```go
// 単一ファイル、最小限の機能
// SOLID原則よりも動作を優先
func main() {
    http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
        // 最小限の実装
    })
    http.ListenAndServe(":8080", nil)
}
```

### Phase 1: 基本的品質向上（SRP部分適用）
**目標**: 最低限の責任分離（1時間）
**SOLID適用度**: 20% - SRPの基本的な適用

```go
// 関数レベルでの責任分離
func handleWebhook(w http.ResponseWriter, r *http.Request) {}
func saveFile(data []byte, filename string) error {}
func validateInput(data map[string]interface{}) error {}
```

### Phase 2: 必要な抽象化（OCP適用）
**目標**: 拡張性が必要になった箇所のみ抽象化（1時間）  
**SOLID適用度**: 40% - SRP完全 + OCP部分適用

```go
// 必要性が明確になったもののみインターフェース化
type FileStorage interface {
    Save(data []byte, filename string) error
}

// まだ必要ない抽象化は行わない
// type Validator interface {} // ❌ まだ不要
```

### Phase 3: 完全なSOLID適用（必要に応じて）
**目標**: テスタビリティとモジュール性が必要になったら
**SOLID適用度**: 100% - 全SOLID原則の適用

```go
// この段階でようやく完全な抽象化
type FileWriter interface {
    Write(ctx context.Context, payload *WebhookPayload) (string, error)
}

type Validator interface {
    Validate(payload *WebhookPayload) error
}

// 依存性注入
func NewWebhookHandler(fw FileWriter, v Validator) *WebhookHandler
```

## 段階的プロジェクト構造の進化

### Phase 0: 最小構造
```
hokku/
└── main.go                     # 全機能をここに（50行程度）
```

### Phase 1: 基本的な分離  
```
hokku/
├── main.go                     # エントリーポイント
├── handlers.go                 # ハンドラ関数
└── utils.go                    # ヘルパー関数
```

### Phase 2: 必要な抽象化後
```
hokku/
├── main.go                     # エントリーポイント  
├── webhook.go                  # Webhook処理
├── storage.go                  # ファイル保存
└── validation.go               # バリデーション（必要になったら）
```

### Phase 3: 完全なSOLID構造（必要性が明確になったら）
```
hokku/
├── cmd/
│   └── hokku/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── handler/
│   │   └── webhook.go          # Webhookハンドラー
│   ├── service/
│   │   ├── interfaces.go       # サービスインターフェース（必要な箇所のみ）
│   │   ├── filestore.go        # ファイル保存実装
│   │   └── validator.go        # バリデーション実装
│   └── model/
│       └── webhook.go          # データモデル
└── test/                       # テスト（必要に応じて）
```

## YAGNI原則の徹底適用

### ❌ 最初から実装してはいけないもの
```go
// ❌ 早すぎる抽象化
type Storage interface {
    Save(data []byte) error
}
type FileStorage struct{}
type S3Storage struct{}  // 使わないのに定義

// ❌ 過剰な設定可能性  
type Config struct {
    Port                int
    StoragePath         string
    MaxFileSize         int64
    MinFileSize         int64    // 使わない
    AllowedExtensions   []string // 使わない
    RetryCount          int      // 使わない
    // ... 50個の設定項目
}

// ❌ 将来の拡張性への過剰な配慮
type Plugin interface {
    Process(data []byte) ([]byte, error)
}
```

### ✅ 段階的実装（必要になってから）
```go
// Phase 0: 直接実装
func saveToFile(data []byte, filename string) error {
    return os.WriteFile(filename, data, 0644)
}

// Phase 1: 基本的な設定
port := "8080"
storagePath := "./storage"

// Phase 2: 必要性が明確になったら設定構造体
type Config struct {
    Port        string  // 実際に使用
    StoragePath string  // 実際に使用
    // 他は必要になってから追加
}

// Phase 3: 本当に必要になったらインターフェース
type FileStorage interface {
    Save(data []byte, filename string) error
}
```

## 段階的実装例

### Phase 0実装例（10行）
```go
http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    os.WriteFile("webhook.json", body, 0644)
    w.WriteHeader(200)
})
```

### Phase 1実装例（+5行のエラーハンドリング）
```go
http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Cannot read", 400)
        return
    }
    if err := os.WriteFile("webhook.json", body, 0644); err != nil {
        http.Error(w, "Cannot save", 500)
        return
    }
    w.WriteHeader(200)
})
```

### Phase 2実装例（+5行のJSONバリデーション）
```go
var data map[string]interface{}
if err := json.Unmarshal(body, &data); err != nil {
    http.Error(w, "Invalid JSON", 400)
    return
}
```

### 2.2 エラーハンドリング（Googleスタイル準拠）
```go
// pkg/errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

// センチネルエラー（Googleスタイル推奨）
var (
    ErrInvalidPayload    = errors.New("invalid payload")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrInsufficientSpace = errors.New("insufficient disk space")
    ErrFileExists        = errors.New("file already exists")
    ErrInvalidPath       = errors.New("invalid file path")
)

// エラーラッピング用ヘルパー（%wを末尾に配置）
func WrapValidationError(field string, err error) error {
    return fmt.Errorf("validation failed for field %s: %w", field, err)
}

func WrapFileError(operation, path string, err error) error {
    return fmt.Errorf("file %s failed for %s: %w", operation, path, err)
}
```

## 3. 依存性注入（コンストラクター関数）

### 3.1 アプリケーション構成
```go
// internal/app/app.go  
package app

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    
    "hokku/internal/config"
    "hokku/internal/handler"
    "hokku/internal/service"
)

type App struct {
    config    *config.Config
    logger    *zap.Logger
    router    *gin.Engine
    
    // インターフェースに依存 (DIP)
    fileStore service.FileStore
    validator service.PayloadValidator
    health    service.HealthChecker
}

// New はDIコンテナの役割を果たす
func New() (*App, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("config load: %w", err)
    }
    
    logger, err := setupLogger(cfg.Logging)
    if err != nil {
        return nil, fmt.Errorf("logger setup: %w", err)
    }
    
    // 具象型を作成してインターフェースに代入
    fileStore := service.NewFileStore(cfg.Storage, logger)
    validator := service.NewValidator(cfg.Validation)
    health := service.NewHealthChecker(cfg)
    
    app := &App{
        config:    cfg,
        logger:    logger,
        fileStore: fileStore,
        validator: validator,
        health:    health,
    }
    
    app.setupRouter()
    return app, nil
}

func (a *App) setupRouter() {
    if a.config.App.Env == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.New()
    
    // ミドルウェア
    r.Use(a.requestIDMiddleware())
    r.Use(a.loggingMiddleware())
    r.Use(a.recoveryMiddleware())
    
    // ハンドラー（インターフェースを渡す）
    webhookHandler := handler.NewWebhookHandler(a.fileStore, a.validator, a.logger)
    healthHandler := handler.NewHealthHandler(a.health)
    
    // ルート設定
    r.GET("/health", healthHandler.Check)
    
    api := r.Group("/")
    if a.config.Auth.Enabled {
        api.Use(a.authMiddleware())
    }
    api.POST("/webhook", webhookHandler.Handle)
    
    a.router = r
}
```

## 4. ハンドラー実装（ISP準拠）

### 4.1 Webhookハンドラー
```go  
// internal/handler/webhook.go
package handler

import (
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
    
    "hokku/internal/model"
    "hokku/internal/service"
    "hokku/pkg/errors"
)

type WebhookHandler struct {
    // インターフェースに依存（DIP）
    fileStore service.FileStore
    validator service.PayloadValidator
    logger    *zap.Logger
}

// NewWebhookHandler はコンストラクター関数
func NewWebhookHandler(
    fs service.FileStore,
    v service.PayloadValidator, 
    logger *zap.Logger,
) *WebhookHandler {
    return &WebhookHandler{
        fileStore: fs,
        validator: v,
        logger:    logger,
    }
}

// Handle は単一の責任（webhook処理）を持つ（SRP）
func (h *WebhookHandler) Handle(c *gin.Context) {
    var payload model.WebhookPayload
    
    // JSONバインド
    if err := c.ShouldBindJSON(&payload); err != nil {
        h.logger.Error("JSON bind failed", zap.Error(err))
        SendValidationError(c, "Invalid JSON format", err.Error())
        return
    }
    
    // バリデーション（責任の分離）
    if err := h.validator.Validate(&payload); err != nil {
        h.logger.Error("Validation failed", zap.Error(err))
        SendValidationError(c, "Validation failed", err.Error())
        return
    }
    
    // システム値設定
    payload.UUID = uuid.New()
    payload.Timestamp = time.Now()
    
    // ファイル保存（責任の分離）
    filePath, err := h.fileStore.Write(&payload)
    if err != nil {
        h.logger.Error("File write failed", 
            zap.Error(err),
            zap.String("uuid", payload.UUID.String()))
        
        // エラーの種類に応じた適切なレスポンス
        SendFileError(c, err)
        return
    }
    
    h.logger.Info("File saved successfully",
        zap.String("path", filePath),
        zap.String("uuid", payload.UUID.String()))
    
    SendSuccess(c, gin.H{
        "message": "File saved successfully",
        "uuid":    payload.UUID.String(),
        "path":    filePath,
    })
}
```

### 4.2 レスポンスヘルパー（Googleスタイル準拠）
```go
// internal/handler/response.go
package handler

import (
    "errors"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    apperrors "hokku/pkg/errors"
)

// SendSuccess は成功レスポンスを送信
func SendSuccess(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "success":    true,
        "data":       data,
        "timestamp":  time.Now(),
        "request_id": c.GetString("request_id"),
    })
}

// SendValidationError はバリデーションエラーを送信
func SendValidationError(c *gin.Context, message, detail string) {
    c.JSON(http.StatusBadRequest, gin.H{
        "success":    false,
        "error":      message,
        "details":    []string{detail},
        "timestamp":  time.Now(),
        "request_id": c.GetString("request_id"),
    })
    c.Abort()
}

// SendFileError はファイル操作エラーを適切なステータスコードで送信
func SendFileError(c *gin.Context, err error) {
    var statusCode int
    var message string
    
    // センチネルエラーによる分岐（Googleスタイル推奨）
    switch {
    case errors.Is(err, apperrors.ErrInsufficientSpace):
        statusCode = http.StatusInsufficientStorage
        message = "Insufficient disk space"
    case errors.Is(err, apperrors.ErrFileExists):
        statusCode = http.StatusConflict
        message = "File already exists"
    case errors.Is(err, apperrors.ErrInvalidPath):
        statusCode = http.StatusBadRequest
        message = "Invalid file path"
    default:
        statusCode = http.StatusInternalServerError
        message = "Internal server error"
    }
    
    c.JSON(statusCode, gin.H{
        "success":    false,
        "error":      message,
        "timestamp":  time.Now(),
        "request_id": c.GetString("request_id"),
    })
    c.Abort()
}
```

## 5. サービス実装（SRP準拠）

### 5.1 ファイルストア実装
```go
// internal/service/filestore.go
package service

import (
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    
    "go.uber.org/zap"
    
    "hokku/internal/config"
    "hokku/internal/model"
    "hokku/pkg/errors"
    "hokku/pkg/security"
)

// fileStore はFileStoreインターフェースの実装
type fileStore struct {
    config config.StorageConfig
    logger *zap.Logger
}

// NewFileStore はコンストラクター関数
func NewFileStore(cfg config.StorageConfig, logger *zap.Logger) FileStore {
    return &fileStore{
        config: cfg,
        logger: logger,
    }
}

// Write はファイル書き込みの単一責任を持つ（SRP）
func (fs *fileStore) Write(payload *model.WebhookPayload) (string, error) {
    // パス構築とバリデーション
    fullPath := filepath.Join(fs.config.BaseDir, payload.Path, payload.FileName)
    
    if err := security.ValidatePath(fullPath, fs.config.BaseDir); err != nil {
        return "", errors.WrapFileError("validate", fullPath, errors.ErrInvalidPath)
    }
    
    // ディスク容量チェック
    if err := fs.checkDiskSpace(); err != nil {
        return "", err // 既にラップ済み
    }
    
    // ディレクトリ作成
    dir := filepath.Dir(fullPath)
    if err := os.MkdirAll(dir, os.FileMode(fs.config.DirPermissions)); err != nil {
        return "", errors.WrapFileError("mkdir", dir, err)
    }
    
    // ファイル存在チェック
    if _, err := os.Stat(fullPath); err == nil {
        return "", errors.WrapFileError("check", fullPath, errors.ErrFileExists)
    }
    
    // コンテンツ準備
    content, err := fs.prepareContent(payload)
    if err != nil {
        return "", err // 既にラップ済み
    }
    
    // ファイル書き込み
    if err := os.WriteFile(fullPath, content, os.FileMode(fs.config.FilePermissions)); err != nil {
        return "", errors.WrapFileError("write", fullPath, err)
    }
    
    fs.logger.Info("File written successfully",
        zap.String("path", fullPath),
        zap.String("uuid", payload.UUID.String()))
    
    return fullPath, nil
}

// prepareContent はコンテンツの準備を行う（ヘルパーメソッド）
func (fs *fileStore) prepareContent(payload *model.WebhookPayload) ([]byte, error) {
    content := []byte(payload.Body)
    
    if payload.Encoding == "base64" {
        decoded, err := base64.StdEncoding.DecodeString(payload.Body)
        if err != nil {
            return nil, fmt.Errorf("base64 decode: %w", err)
        }
        content = decoded
    }
    
    return content, nil
}

// checkDiskSpace は簡単なディスク容量チェック
func (fs *fileStore) checkDiskSpace() error {
    // 実際の実装では statfs syscall を使用
    // ここでは簡略化
    return nil
}
```

### 5.2 バリデーター実装
```go
// internal/service/validator.go
package service

import (
    "strings"
    
    "hokku/internal/config"
    "hokku/internal/model"
    "hokku/pkg/errors"
)

type validator struct {
    config config.ValidationConfig
}

func NewValidator(cfg config.ValidationConfig) PayloadValidator {
    return &validator{config: cfg}
}

// Validate はペイロード検証の単一責任を持つ（SRP）
func (v *validator) Validate(payload *model.WebhookPayload) error {
    // 必須フィールドチェック
    if payload.Title == "" {
        return errors.WrapValidationError("title", errors.ErrInvalidPayload)
    }
    if payload.FileName == "" {
        return errors.WrapValidationError("filename", errors.ErrInvalidPayload)
    }
    if payload.Body == "" {
        return errors.WrapValidationError("body", errors.ErrInvalidPayload)
    }
    
    // ファイル名検証
    if err := v.validateFileName(payload.FileName); err != nil {
        return errors.WrapValidationError("filename", err)
    }
    
    // 拡張子チェック
    if err := v.validateExtension(payload.FileName); err != nil {
        return errors.WrapValidationError("filename", err)
    }
    
    return nil
}

func (v *validator) validateFileName(filename string) error {
    // 危険な文字チェック
    dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
    for _, char := range dangerous {
        if strings.Contains(filename, char) {
            return errors.ErrInvalidPath
        }
    }
    return nil
}

func (v *validator) validateExtension(filename string) error {
    // 許可された拡張子のチェック
    for _, ext := range v.config.AllowedExtensions {
        if strings.HasSuffix(filename, ext) {
            return nil
        }
    }
    return errors.ErrInvalidPayload
}
```

## 6. テスト戦略（SOLID準拠）

### 6.1 インターフェースベースのテスト
```go
// test/handler/webhook_test.go
package handler_test

import (
    "bytes"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "go.uber.org/zap/zaptest"
    
    "hokku/internal/handler"
    "hokku/internal/model"
    "hokku/test/mocks"
)

func TestWebhookHandler_Handle(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    tests := []struct {
        name           string
        payload        map[string]interface{}
        setupMocks     func(*mocks.FileStore, *mocks.PayloadValidator)
        expectedStatus int
    }{
        {
            name: "成功ケース",
            payload: map[string]interface{}{
                "title":    "Test",
                "filename": "test.md", 
                "body":     "content",
            },
            setupMocks: func(fs *mocks.FileStore, v *mocks.PayloadValidator) {
                v.On("Validate", mock.Anything).Return(nil)
                fs.On("Write", mock.Anything).Return("/path/to/file", nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name: "バリデーションエラー",
            payload: map[string]interface{}{
                "title": "", // 空のtitle
            },
            setupMocks: func(fs *mocks.FileStore, v *mocks.PayloadValidator) {
                v.On("Validate", mock.Anything).Return(errors.New("validation error"))
            },
            expectedStatus: http.StatusBadRequest,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // モック作成
            mockFileStore := new(mocks.FileStore)
            mockValidator := new(mocks.PayloadValidator)
            
            // モック設定
            tt.setupMocks(mockFileStore, mockValidator)
            
            // ハンドラー作成
            logger := zaptest.NewLogger(t)
            h := handler.NewWebhookHandler(mockFileStore, mockValidator, logger)
            
            // リクエスト作成
            body, _ := json.Marshal(tt.payload)
            req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
            req.Header.Set("Content-Type", "application/json")
            
            w := httptest.NewRecorder()
            c, _ := gin.CreateTestContext(w)
            c.Request = req
            
            // 実行
            h.Handle(c)
            
            // 検証
            assert.Equal(t, tt.expectedStatus, w.Code)
            mockFileStore.AssertExpectations(t)
            mockValidator.AssertExpectations(t)
        })
    }
}
```

## 7. YAGNI原則の適用

### 7.1 実装しない機能（YAGNI）
- OAuth2認証（APIキーで十分）
- メトリクス収集（ログで十分）
- データベース（ファイルシステムで十分） 
- キューイングシステム（同期処理で十分）
- 複雑な設定システム（Viperで十分）

### 7.2 最小限の機能のみ実装
- Webhook受信
- ファイル保存
- 基本認証
- 構造化ログ
- ヘルスチェック

## 8. Makefile（品質重視）

```makefile
# Makefile
.PHONY: help build run test test-unit test-integration lint fmt clean

BINARY_NAME=hokku
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -s -w"

help: ## Display help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: fmt lint test ## Build binary
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/hokku/main.go

run: ## Run application  
	go run cmd/hokku/main.go

test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	go test -v -race -cover ./internal/...

test-integration: ## Run integration tests  
	go test -v -race -cover ./test/integration/...

lint: ## Run linter
	golangci-lint run --timeout=5m

fmt: ## Format code
	go fmt ./...
	goimports -w .

clean: ## Clean artifacts
	rm -rf bin/ *.out *.html

.DEFAULT_GOAL := help
```

## まとめ

この設計は以下のSOLID/YAGNI原則に準拠：

### ✅ SOLID原則準拠
1. **SRP**: 各構造体が単一責任
2. **OCP**: インターフェースによる拡張性 
3. **LSP**: インターフェース実装で契約維持
4. **ISP**: 小さく特化したインターフェース
5. **DIP**: インターフェースに依存

### ✅ YAGNI原則準拠
- 必要最小限の機能のみ実装
- 将来の要件を推測せず現在必要な機能に集中
- 過度な抽象化を避ける

### ✅ Goベストプラクティス準拠
- Googleスタイルガイドに従ったエラーハンドリング
- センチネルエラーの使用
- 適切なインターフェース配置
- コンストラクター関数の使用

この設計により、保守性と拡張性を保ちながら、シンプルで実装しやすいアーキテクチャを実現できます。