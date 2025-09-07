# Hokku - 改善された設計仕様書

## 概要
SPECIFICATION.mdをベースに、Gin/Go Clean Architectureのベストプラクティスを適用した改善設計です。

## 1. プロジェクト構造の最適化

### 1.1 Clean Architecture層構造
```
hokku/
├── cmd/
│   └── hokku/
│       └── main.go              # エントリーポイント（DI設定）
├── internal/
│   ├── domain/                 # ビジネスドメイン層（依存性なし）
│   │   ├── entities/           
│   │   │   ├── webhook.go      # Webhookエンティティ
│   │   │   └── file.go         # Fileエンティティ
│   │   ├── repositories/       # リポジトリインターフェース
│   │   │   └── webhook_repository.go
│   │   └── services/            # ドメインサービス
│   │       └── webhook_service.go
│   ├── application/             # アプリケーション層
│   │   ├── usecases/           
│   │   │   ├── save_webhook.go # ユースケース実装
│   │   │   └── interfaces.go   # ユースケースインターフェース
│   │   └── dto/                # データ転送オブジェクト
│   │       ├── request.go      
│   │       └── response.go     
│   ├── infrastructure/          # インフラストラクチャ層
│   │   ├── config/             
│   │   │   ├── config.go       # Viper設定管理
│   │   │   └── env.go          # 環境変数処理
│   │   ├── persistence/        # 永続化実装
│   │   │   └── file_repository.go
│   │   ├── server/             # HTTPサーバー
│   │   │   ├── router.go       # Ginルーター設定
│   │   │   └── middleware/     # ミドルウェア
│   │   │       ├── auth.go     
│   │   │       ├── recovery.go # カスタムリカバリー
│   │   │       ├── logger.go   
│   │   │       └── validator.go
│   │   └── logger/             # Zapロガー設定
│   │       └── logger.go       
│   └── interfaces/             # インターフェース層
│       ├── api/                # APIハンドラー
│       │   ├── v1/             # バージョニング
│       │   │   ├── webhook_handler.go
│       │   │   └── health_handler.go
│       │   └── errors/         # エラーハンドリング
│       │       ├── api_error.go
│       │       └── handler.go
│       └── validators/         # バリデーション
│           └── webhook_validator.go
├── pkg/                        # 共有パッケージ
│   ├── errors/                # カスタムエラー定義
│   │   ├── errors.go          
│   │   └── codes.go           # エラーコード定義
│   ├── security/              # セキュリティユーティリティ
│   │   ├── path_validator.go  
│   │   └── sanitizer.go       
│   └── utils/                 # 汎用ユーティリティ
│       ├── disk.go            
│       └── uuid.go            
├── di/                        # 依存性注入設定
│   └── wire.go                # Wire設定（またはFx）
├── migrations/                # データベースマイグレーション（将来用）
├── scripts/                   # ユーティリティスクリプト
├── test/                      
│   ├── unit/                  
│   ├── integration/           
│   ├── e2e/                   # E2Eテスト
│   └── fixtures/              # テストフィクスチャ
└── build/                     # ビルド設定
    ├── docker/                
    │   └── Dockerfile         
    └── ci/                    
        └── .github/           
            └── workflows/     
```

## 2. エラーハンドリングパターン

### 2.1 カスタムエラー型定義
```go
// pkg/errors/errors.go
package errors

import (
    "fmt"
    "net/http"
)

type ErrorCode string

const (
    ErrCodeValidation      ErrorCode = "VALIDATION_ERROR"
    ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
    ErrCodeNotFound        ErrorCode = "NOT_FOUND"
    ErrCodeDiskSpace       ErrorCode = "DISK_SPACE_ERROR"
    ErrCodeFileExists      ErrorCode = "FILE_EXISTS"
    ErrCodePathTraversal   ErrorCode = "PATH_TRAVERSAL"
    ErrCodeInternal        ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
    Code       ErrorCode   `json:"code"`
    Message    string      `json:"message"`
    StatusCode int         `json:"-"`
    Details    interface{} `json:"details,omitempty"`
    Err        error       `json:"-"`
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
    return e.Err
}

// 事前定義エラー
var (
    ErrUnauthorized = &AppError{
        Code:       ErrCodeUnauthorized,
        Message:    "Unauthorized access",
        StatusCode: http.StatusUnauthorized,
    }
    
    ErrDiskSpaceInsufficient = &AppError{
        Code:       ErrCodeDiskSpace,
        Message:    "Insufficient disk space",
        StatusCode: http.StatusInsufficientStorage,
    }
)
```

### 2.2 エラーハンドラー
```go
// internal/interfaces/api/errors/handler.go
package errors

import (
    "errors"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    apperrors "hokku/pkg/errors"
)

func HandleError(logger *zap.Logger, c *gin.Context, err error) {
    var appErr *apperrors.AppError
    
    if errors.As(err, &appErr) {
        logger.Error("Application error",
            zap.String("code", string(appErr.Code)),
            zap.Error(appErr.Err),
            zap.String("path", c.Request.URL.Path),
        )
        
        c.JSON(appErr.StatusCode, gin.H{
            "success": false,
            "error": gin.H{
                "code":    appErr.Code,
                "message": appErr.Message,
                "details": appErr.Details,
            },
            "request_id": c.GetString("request_id"),
        })
        return
    }
    
    // 未処理エラー
    logger.Error("Unhandled error",
        zap.Error(err),
        zap.String("path", c.Request.URL.Path),
    )
    
    c.JSON(http.StatusInternalServerError, gin.H{
        "success": false,
        "error": gin.H{
            "code":    apperrors.ErrCodeInternal,
            "message": "An internal error occurred",
        },
        "request_id": c.GetString("request_id"),
    })
}
```

## 3. 依存性注入（DI）設計

### 3.1 Wire使用例
```go
// di/wire.go
//go:build wireinject
// +build wireinject

package di

import (
    "github.com/google/wire"
    "hokku/internal/domain/repositories"
    "hokku/internal/domain/services"
    "hokku/internal/application/usecases"
    "hokku/internal/infrastructure/config"
    "hokku/internal/infrastructure/logger"
    "hokku/internal/infrastructure/persistence"
    "hokku/internal/infrastructure/server"
    "hokku/internal/interfaces/api/v1"
)

func InitializeApp() (*App, error) {
    wire.Build(
        // Infrastructure
        config.NewConfig,
        logger.NewLogger,
        persistence.NewFileRepository,
        server.NewRouter,
        
        // Domain
        services.NewWebhookService,
        
        // Application
        usecases.NewSaveWebhookUseCase,
        
        // Interfaces
        v1.NewWebhookHandler,
        v1.NewHealthHandler,
        
        // Bind interfaces
        wire.Bind(new(repositories.WebhookRepository), new(*persistence.FileRepository)),
        
        // App
        NewApp,
    )
    
    return nil, nil
}

type App struct {
    Router *server.Router
    Config *config.Config
    Logger *logger.Logger
}
```

### 3.2 コンストラクタパターン
```go
// internal/application/usecases/save_webhook.go
package usecases

import (
    "context"
    "hokku/internal/domain/entities"
    "hokku/internal/domain/repositories"
    "hokku/internal/domain/services"
    "go.uber.org/zap"
)

type SaveWebhookUseCase struct {
    repo    repositories.WebhookRepository
    service *services.WebhookService
    logger  *zap.Logger
}

func NewSaveWebhookUseCase(
    repo repositories.WebhookRepository,
    service *services.WebhookService,
    logger *zap.Logger,
) *SaveWebhookUseCase {
    return &SaveWebhookUseCase{
        repo:    repo,
        service: service,
        logger:  logger,
    }
}

func (uc *SaveWebhookUseCase) Execute(ctx context.Context, input *SaveWebhookInput) (*SaveWebhookOutput, error) {
    // ビジネスロジック実装
}
```

## 4. ミドルウェア設計

### 4.1 カスタムリカバリーミドルウェア
```go
// internal/infrastructure/server/middleware/recovery.go
package middleware

import (
    "fmt"
    "net/http"
    "runtime/debug"
    
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "hokku/pkg/errors"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        err := fmt.Errorf("%v", recovered)
        
        logger.Error("Panic recovered",
            zap.Any("error", recovered),
            zap.String("stack", string(debug.Stack())),
            zap.String("path", c.Request.URL.Path),
        )
        
        appErr := &errors.AppError{
            Code:       errors.ErrCodeInternal,
            Message:    "Internal server error",
            StatusCode: http.StatusInternalServerError,
        }
        
        c.JSON(appErr.StatusCode, gin.H{
            "success": false,
            "error": gin.H{
                "code":    appErr.Code,
                "message": appErr.Message,
            },
            "request_id": c.GetString("request_id"),
        })
        
        c.Abort()
    })
}
```

### 4.2 認証ミドルウェア改善
```go
// internal/infrastructure/server/middleware/auth.go
package middleware

import (
    "strings"
    "hokku/internal/infrastructure/config"
    "hokku/pkg/errors"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type AuthMiddleware struct {
    config *config.AuthConfig
    logger *zap.Logger
}

func NewAuthMiddleware(config *config.AuthConfig, logger *zap.Logger) *AuthMiddleware {
    return &AuthMiddleware{
        config: config,
        logger: logger,
    }
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !m.config.Enabled {
            c.Next()
            return
        }
        
        apiKey := m.extractAPIKey(c)
        if apiKey == "" {
            m.handleAuthError(c, "Missing API key")
            return
        }
        
        if !m.isValidAPIKey(apiKey) {
            m.handleAuthError(c, "Invalid API key")
            return
        }
        
        c.Set("api_key_name", m.getAPIKeyName(apiKey))
        c.Next()
    }
}

func (m *AuthMiddleware) extractAPIKey(c *gin.Context) string {
    // Headerから取得
    if key := c.GetHeader("X-API-Key"); key != "" {
        return key
    }
    
    // Bearer tokenから取得
    if auth := c.GetHeader("Authorization"); auth != "" {
        parts := strings.Split(auth, " ")
        if len(parts) == 2 && parts[0] == "Bearer" {
            return parts[1]
        }
    }
    
    // Queryから取得（非推奨）
    return c.Query("api_key")
}
```

## 5. 設定管理の改善

### 5.1 Viper統合
```go
// internal/infrastructure/config/config.go
package config

import (
    "fmt"
    "time"
    
    "github.com/spf13/viper"
)

type Config struct {
    App      AppConfig      `mapstructure:"app"`
    Server   ServerConfig   `mapstructure:"server"`
    Storage  StorageConfig  `mapstructure:"storage"`
    Auth     AuthConfig     `mapstructure:"auth"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

func NewConfig() (*Config, error) {
    v := viper.New()
    
    // 設定ファイルパス
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath("./config")
    v.AddConfigPath(".")
    
    // 環境変数のプレフィックス
    v.SetEnvPrefix("HOKKU")
    v.AutomaticEnv()
    
    // デフォルト値設定
    setDefaults(v)
    
    // 設定ファイル読み込み
    if err := v.ReadInConfig(); err != nil {
        // 設定ファイルがない場合はデフォルト値と環境変数のみ使用
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }
    
    var config Config
    if err := v.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // バリデーション
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return &config, nil
}

func setDefaults(v *viper.Viper) {
    v.SetDefault("app.env", "development")
    v.SetDefault("server.port", 20023)
    v.SetDefault("server.read_timeout", "30s")
    v.SetDefault("server.write_timeout", "30s")
    v.SetDefault("storage.base_dir", "./storage")
    v.SetDefault("storage.file_permissions", 0644)
    v.SetDefault("storage.dir_permissions", 0755)
    v.SetDefault("logging.level", "info")
    v.SetDefault("logging.format", "json")
}
```

## 6. テスト戦略の詳細

### 6.1 ユニットテスト構造
```go
// internal/application/usecases/save_webhook_test.go
package usecases_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/zap/zaptest"
    
    "hokku/internal/application/usecases"
    "hokku/test/mocks"
)

type SaveWebhookUseCaseTestSuite struct {
    suite.Suite
    useCase  *usecases.SaveWebhookUseCase
    mockRepo *mocks.MockWebhookRepository
    logger   *zap.Logger
}

func (suite *SaveWebhookUseCaseTestSuite) SetupTest() {
    suite.logger = zaptest.NewLogger(suite.T())
    suite.mockRepo = new(mocks.MockWebhookRepository)
    suite.useCase = usecases.NewSaveWebhookUseCase(
        suite.mockRepo,
        suite.logger,
    )
}

func (suite *SaveWebhookUseCaseTestSuite) TestExecute_Success() {
    // Arrange
    ctx := context.Background()
    input := &usecases.SaveWebhookInput{
        Title:    "Test",
        FileName: "test.md",
        Body:     "content",
    }
    
    suite.mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
    
    // Act
    output, err := suite.useCase.Execute(ctx, input)
    
    // Assert
    assert.NoError(suite.T(), err)
    assert.NotNil(suite.T(), output)
    suite.mockRepo.AssertExpectations(suite.T())
}

func TestSaveWebhookUseCaseTestSuite(t *testing.T) {
    suite.Run(t, new(SaveWebhookUseCaseTestSuite))
}
```

### 6.2 統合テスト
```go
// test/integration/webhook_test.go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "hokku/di"
)

func TestWebhookEndpoint(t *testing.T) {
    // アプリケーション初期化
    app, err := di.InitializeTestApp()
    assert.NoError(t, err)
    
    // テストデータ
    payload := map[string]interface{}{
        "title":    "Test Document",
        "filename": "test.md",
        "body":     "# Test Content",
    }
    
    body, _ := json.Marshal(payload)
    
    // リクエスト作成
    req := httptest.NewRequest(
        http.MethodPost,
        "/api/v1/webhook",
        bytes.NewBuffer(body),
    )
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", "test-key")
    
    // レスポンス記録
    w := httptest.NewRecorder()
    
    // 実行
    app.Router.ServeHTTP(w, req)
    
    // アサーション
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err = json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.True(t, response["success"].(bool))
}
```

## 7. Goroutine安全性

### 7.1 Context管理
```go
// internal/interfaces/api/v1/webhook_handler.go
package v1

import (
    "context"
    "time"
    
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
    // タイムアウト付きコンテキスト
    ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
    defer cancel()
    
    // 非同期処理の場合
    if c.Query("async") == "true" {
        // コンテキストをコピー
        ctxCopy := c.Copy()
        
        go func() {
            // 新しいタイムアウトコンテキスト
            asyncCtx, asyncCancel := context.WithTimeout(
                context.Background(),
                5*time.Minute,
            )
            defer asyncCancel()
            
            if err := h.processAsync(asyncCtx, ctxCopy); err != nil {
                h.logger.Error("Async processing failed",
                    zap.Error(err),
                    zap.String("request_id", ctxCopy.GetString("request_id")),
                )
            }
        }()
        
        c.JSON(http.StatusAccepted, gin.H{
            "success": true,
            "message": "Request accepted for processing",
        })
        return
    }
    
    // 同期処理
    result, err := h.useCase.Execute(ctx, input)
    // ...
}
```

## 8. パフォーマンス最適化

### 8.1 バッファプール使用
```go
// internal/infrastructure/persistence/file_repository.go
package persistence

import (
    "sync"
    "bytes"
)

var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func (r *FileRepository) WriteFile(path string, content []byte) error {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    buf.Write(content)
    // ファイル書き込み処理
}
```

## 9. セキュリティ強化

### 9.1 入力サニタイズ
```go
// pkg/security/sanitizer.go
package security

import (
    "html"
    "regexp"
    "strings"
)

var (
    // 危険な文字パターン
    dangerousPatterns = regexp.MustCompile(`[<>\"'&]`)
    
    // ファイル名として安全な文字のみ
    safeFileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

func SanitizeFileName(name string) (string, error) {
    // HTML特殊文字エスケープ
    name = html.EscapeString(name)
    
    // 危険な文字を除去
    name = dangerousPatterns.ReplaceAllString(name, "")
    
    // 空白をアンダースコアに変換
    name = strings.ReplaceAll(name, " ", "_")
    
    // 長さチェック
    if len(name) > 255 {
        name = name[:255]
    }
    
    // 安全性確認
    if !safeFileNamePattern.MatchString(name) {
        return "", ErrInvalidFileName
    }
    
    return name, nil
}
```

## 10. Makefile改善

```makefile
# Improved Makefile
.PHONY: help build run test clean lint fmt migrate

# Variables
BINARY_NAME=hokku
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -s -w"
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

help: ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@go build ${LDFLAGS} -o $(GOBIN)/$(BINARY_NAME) cmd/hokku/main.go
	@echo "$(GREEN)Build complete!$(NC)"

run: ## Run the application
	@go run cmd/hokku/main.go

test: ## Run tests with coverage
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(GREEN)Tests complete!$(NC)"

test-coverage: test ## Generate coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

lint: ## Run linters
	@echo "$(YELLOW)Running linters...$(NC)"
	@golangci-lint run --timeout=5m
	@echo "$(GREEN)Linting complete!$(NC)"

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@go fmt ./...
	@goimports -w .
	@echo "$(GREEN)Formatting complete!$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(GOBIN) coverage.* vendor/
	@echo "$(GREEN)Clean complete!$(NC)"

install-tools: ## Install development tools
	@echo "$(YELLOW)Installing tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/google/wire/cmd/wire@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Tools installed!$(NC)"

docker-build: ## Build Docker image
	@docker build -t $(BINARY_NAME):$(VERSION) -f build/docker/Dockerfile .

docker-run: ## Run Docker container
	@docker run -p 20023:20023 --env-file .env $(BINARY_NAME):$(VERSION)

.DEFAULT_GOAL := help
```

## まとめ

この改善設計により、以下の利点が得られます：

1. **保守性向上**: Clean Architectureによる層分離で変更影響を最小化
2. **テスタビリティ**: DIによるモック化が容易で、包括的なテストが可能
3. **エラー処理**: 構造化されたエラー型で一貫したエラーレスポンス
4. **セキュリティ**: 多層防御による堅牢なセキュリティ
5. **パフォーマンス**: 効率的なリソース管理とゼロアロケーション
6. **開発効率**: 明確な構造とツールサポート

これらの改善により、本番環境でも安定して動作する高品質なWebhookサービスを構築できます。