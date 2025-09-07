# Hokku - 実用的な設計仕様書

## 概要
SPECIFICATION.mdをベースに、シンプルさと実用性を重視した改善設計です。
過度な抽象化を避け、Go/Ginのベストプラクティスに沿った実装を目指します。

## 1. シンプルなプロジェクト構造

```
hokku/
├── cmd/
│   └── hokku/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go            # 設定管理（Viper）
│   ├── handler/
│   │   ├── webhook.go           # Webhookハンドラー
│   │   └── health.go            # ヘルスチェック
│   ├── middleware/
│   │   ├── auth.go              # 認証ミドルウェア
│   │   ├── logger.go            # ロギングミドルウェア
│   │   └── recovery.go          # リカバリーミドルウェア
│   ├── model/
│   │   ├── webhook.go           # リクエスト/レスポンスモデル
│   │   └── error.go             # エラーレスポンスモデル
│   ├── service/
│   │   ├── file_writer.go       # ファイル書き込みサービス
│   │   └── validator.go         # バリデーションサービス
│   └── router/
│       └── router.go            # ルーター設定
├── pkg/
│   ├── logger/
│   │   └── logger.go            # Zapロガー設定
│   └── security/
│       ├── path.go              # パスバリデーション
│       └── sanitizer.go         # 入力サニタイズ
├── test/
│   ├── handler/                # ハンドラーテスト
│   ├── service/                # サービステスト
│   └── integration/            # 統合テスト
├── config/
│   ├── config.yaml             # デフォルト設定
│   └── config.example.yaml     # 設定例
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 2. 実用的なエラーハンドリング

### 2.1 シンプルなエラーレスポンス
```go
// internal/model/error.go
package model

import "time"

type ErrorResponse struct {
    Success   bool      `json:"success"`
    Error     string    `json:"error"`
    Code      string    `json:"code,omitempty"`
    Details   []string  `json:"details,omitempty"`
    Timestamp time.Time `json:"timestamp"`
    RequestID string    `json:"request_id,omitempty"`
}

// よく使うエラーを定数化
const (
    ErrCodeValidation    = "VALIDATION_ERROR"
    ErrCodeUnauthorized  = "UNAUTHORIZED"
    ErrCodeNotFound      = "NOT_FOUND"
    ErrCodeDiskSpace     = "DISK_SPACE_ERROR"
    ErrCodeFileExists    = "FILE_EXISTS"
    ErrCodeInternal      = "INTERNAL_ERROR"
)
```

### 2.2 エラーハンドリングヘルパー
```go
// internal/handler/helper.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "hokku/internal/model"
    "time"
)

func SendError(c *gin.Context, status int, code, message string, details ...string) {
    c.JSON(status, model.ErrorResponse{
        Success:   false,
        Error:     message,
        Code:      code,
        Details:   details,
        Timestamp: time.Now(),
        RequestID: c.GetString("request_id"),
    })
    c.Abort()
}

func SendSuccess(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    data,
        "timestamp": time.Now(),
        "request_id": c.GetString("request_id"),
    })
}
```

## 3. 直接的な依存性注入（DIコンテナなし）

### 3.1 App構造体による依存管理
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
    Config      *config.Config
    Logger      *zap.Logger
    Router      *gin.Engine
    FileWriter  *service.FileWriter
    Validator   *service.Validator
}

func New() (*App, error) {
    // 設定読み込み
    cfg, err := config.Load()
    if err != nil {
        return nil, err
    }
    
    // ロガー初期化
    logger, err := setupLogger(cfg.Logging)
    if err != nil {
        return nil, err
    }
    
    // サービス初期化
    fileWriter := service.NewFileWriter(cfg.Storage, logger)
    validator := service.NewValidator(cfg.Validation)
    
    app := &App{
        Config:     cfg,
        Logger:     logger,
        FileWriter: fileWriter,
        Validator:  validator,
    }
    
    // ルーター設定
    app.setupRouter()
    
    return app, nil
}

func (a *App) setupRouter() {
    if a.Config.App.Env == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.New()
    
    // ミドルウェア設定
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger(a.Logger))
    r.Use(middleware.Recovery(a.Logger))
    
    // ハンドラー初期化
    webhookHandler := handler.NewWebhookHandler(
        a.FileWriter,
        a.Validator,
        a.Logger,
    )
    healthHandler := handler.NewHealthHandler(a.Config)
    
    // ルート設定
    r.GET("/health", healthHandler.Check)
    
    // 認証が必要なエンドポイント
    api := r.Group("/")
    if a.Config.Auth.Enabled {
        api.Use(middleware.Auth(a.Config.Auth, a.Logger))
    }
    api.POST("/webhook", webhookHandler.Handle)
    
    a.Router = r
}
```

## 4. 実用的なミドルウェア

### 4.1 シンプルな認証ミドルウェア
```go
// internal/middleware/auth.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "hokku/internal/config"
    "hokku/internal/handler"
    "hokku/internal/model"
    "net/http"
    "strings"
)

func Auth(cfg config.AuthConfig, logger *zap.Logger) gin.HandlerFunc {
    // 有効なAPIキーをマップに変換（起動時に1回だけ）
    validKeys := make(map[string]string)
    for _, key := range cfg.APIKeys {
        if key.Enabled {
            validKeys[key.Key] = key.Name
        }
    }
    
    return func(c *gin.Context) {
        // APIキー取得（優先順位: Header > Bearer > Query）
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
                apiKey = strings.TrimPrefix(auth, "Bearer ")
            }
        }
        if apiKey == "" {
            apiKey = c.Query("api_key")  // 非推奨だが互換性のため
        }
        
        // 検証
        keyName, valid := validKeys[apiKey]
        if !valid {
            handler.SendError(c, http.StatusUnauthorized, 
                model.ErrCodeUnauthorized, 
                "Invalid or missing API key")
            return
        }
        
        // コンテキストに保存
        c.Set("api_key_name", keyName)
        c.Next()
    }
}
```

### 4.2 リカバリーミドルウェア
```go
// internal/middleware/recovery.go
package middleware

import (
    "fmt"
    "net/http"
    "runtime/debug"
    
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "hokku/internal/handler"
    "hokku/internal/model"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        // スタックトレースをログ出力
        logger.Error("Panic recovered",
            zap.Any("error", recovered),
            zap.String("stack", string(debug.Stack())),
            zap.String("path", c.Request.URL.Path),
        )
        
        // クライアントには詳細を隠す
        handler.SendError(c, 
            http.StatusInternalServerError,
            model.ErrCodeInternal,
            "An internal error occurred")
    })
}
```

## 5. コア機能の実装

### 5.1 Webhookハンドラー
```go
// internal/handler/webhook.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
    "hokku/internal/model"
    "hokku/internal/service"
    "time"
)

type WebhookHandler struct {
    fileWriter *service.FileWriter
    validator  *service.Validator
    logger     *zap.Logger
}

func NewWebhookHandler(fw *service.FileWriter, v *service.Validator, l *zap.Logger) *WebhookHandler {
    return &WebhookHandler{
        fileWriter: fw,
        validator:  v,
        logger:     l,
    }
}

func (h *WebhookHandler) Handle(c *gin.Context) {
    var payload model.WebhookPayload
    
    // JSONバインド
    if err := c.ShouldBindJSON(&payload); err != nil {
        SendError(c, http.StatusBadRequest, 
            model.ErrCodeValidation,
            "Invalid JSON payload",
            err.Error())
        return
    }
    
    // バリデーション
    if err := h.validator.ValidatePayload(&payload); err != nil {
        SendError(c, http.StatusBadRequest,
            model.ErrCodeValidation,
            "Validation failed",
            err.Error())
        return
    }
    
    // UUID生成
    payload.UUID = uuid.New()
    payload.Timestamp = time.Now()
    
    // ファイル保存
    filePath, err := h.fileWriter.Write(&payload)
    if err != nil {
        // エラーの種類に応じて適切なステータスコードを返す
        status, code := h.mapError(err)
        SendError(c, status, code, err.Error())
        return
    }
    
    // 成功レスポンス
    SendSuccess(c, gin.H{
        "message": "File saved successfully",
        "uuid":    payload.UUID.String(),
        "path":    filePath,
    })
}

func (h *WebhookHandler) mapError(err error) (int, string) {
    // エラーメッセージから適切なステータスコードとエラーコードを判定
    switch {
    case contains(err, "disk space"):
        return http.StatusInsufficientStorage, model.ErrCodeDiskSpace
    case contains(err, "already exists"):
        return http.StatusConflict, model.ErrCodeFileExists
    case contains(err, "invalid path"):
        return http.StatusBadRequest, model.ErrCodeValidation
    default:
        return http.StatusInternalServerError, model.ErrCodeInternal
    }
}
```

### 5.2 ファイル書き込みサービス
```go
// internal/service/file_writer.go
package service

import (
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    
    "go.uber.org/zap"
    "hokku/internal/config"
    "hokku/internal/model"
    "hokku/pkg/security"
)

type FileWriter struct {
    config config.StorageConfig
    logger *zap.Logger
}

func NewFileWriter(cfg config.StorageConfig, logger *zap.Logger) *FileWriter {
    return &FileWriter{
        config: cfg,
        logger: logger,
    }
}

func (fw *FileWriter) Write(payload *model.WebhookPayload) (string, error) {
    // パス構築
    fullPath := filepath.Join(
        fw.config.BaseDir,
        payload.Path,
        payload.FileName,
    )
    
    // セキュリティチェック
    if err := security.ValidatePath(fullPath, fw.config.BaseDir); err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    // ディスク容量チェック（シンプルな実装）
    if err := fw.checkDiskSpace(); err != nil {
        return "", err
    }
    
    // ディレクトリ作成
    dir := filepath.Dir(fullPath)
    if err := os.MkdirAll(dir, os.FileMode(fw.config.DirPermissions)); err != nil {
        return "", fmt.Errorf("failed to create directory: %w", err)
    }
    
    // ファイル存在チェック
    if _, err := os.Stat(fullPath); err == nil {
        return "", fmt.Errorf("file already exists: %s", fullPath)
    }
    
    // コンテンツ準備
    content := []byte(payload.Body)
    if payload.Encoding == "base64" {
        decoded, err := base64.StdEncoding.DecodeString(payload.Body)
        if err != nil {
            return "", fmt.Errorf("base64 decode failed: %w", err)
        }
        content = decoded
    }
    
    // ファイル書き込み
    if err := os.WriteFile(fullPath, content, os.FileMode(fw.config.FilePermissions)); err != nil {
        return "", fmt.Errorf("failed to write file: %w", err)
    }
    
    fw.logger.Info("File written successfully",
        zap.String("path", fullPath),
        zap.String("uuid", payload.UUID.String()),
    )
    
    return fullPath, nil
}
```

## 6. 設定管理（Viper使用）

```go
// internal/config/config.go
package config

import (
    "github.com/spf13/viper"
    "time"
)

type Config struct {
    App      AppConfig      `mapstructure:"app"`
    Server   ServerConfig   `mapstructure:"server"`
    Storage  StorageConfig  `mapstructure:"storage"`
    Auth     AuthConfig     `mapstructure:"auth"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

type AppConfig struct {
    Name    string `mapstructure:"name"`
    Version string `mapstructure:"version"`
    Env     string `mapstructure:"env"`
}

type ServerConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type StorageConfig struct {
    BaseDir         string `mapstructure:"base_dir"`
    DiskQuota       int64  `mapstructure:"disk_quota"`
    FilePermissions uint32 `mapstructure:"file_permissions"`
    DirPermissions  uint32 `mapstructure:"dir_permissions"`
}

type AuthConfig struct {
    Enabled bool     `mapstructure:"enabled"`
    APIKeys []APIKey `mapstructure:"api_keys"`
}

type APIKey struct {
    Key     string `mapstructure:"key"`
    Name    string `mapstructure:"name"`
    Enabled bool   `mapstructure:"enabled"`
}

type LoggingConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
    v := viper.New()
    
    // 設定ファイル
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath("./config")
    v.AddConfigPath(".")
    
    // 環境変数
    v.SetEnvPrefix("HOKKU")
    v.AutomaticEnv()
    
    // デフォルト値
    v.SetDefault("app.env", "development")
    v.SetDefault("server.port", 20023)
    v.SetDefault("server.read_timeout", "30s")
    v.SetDefault("server.write_timeout", "30s")
    v.SetDefault("storage.base_dir", "./storage")
    
    // 読み込み
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }
    
    var config Config
    if err := v.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## 7. テスト戦略

### 7.1 ハンドラーテスト（httptest使用）
```go
// test/handler/webhook_test.go
package handler_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "hokku/internal/handler"
    "hokku/internal/service"
    "hokku/test/helpers"
)

func TestWebhookHandler_Handle(t *testing.T) {
    // セットアップ
    gin.SetMode(gin.TestMode)
    app := helpers.SetupTestApp(t)
    
    tests := []struct {
        name       string
        payload    map[string]interface{}
        apiKey     string
        wantStatus int
    }{
        {
            name: "正常なリクエスト",
            payload: map[string]interface{}{
                "title":    "Test",
                "filename": "test.md",
                "body":     "content",
            },
            apiKey:     "test-key",
            wantStatus: http.StatusOK,
        },
        {
            name: "不正なJSON",
            payload: map[string]interface{}{
                "title": "Test",
                // filename欠落
            },
            apiKey:     "test-key",
            wantStatus: http.StatusBadRequest,
        },
        {
            name: "認証エラー",
            payload: map[string]interface{}{
                "title":    "Test",
                "filename": "test.md",
                "body":     "content",
            },
            apiKey:     "invalid-key",
            wantStatus: http.StatusUnauthorized,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // リクエスト作成
            body, _ := json.Marshal(tt.payload)
            req := httptest.NewRequest(
                http.MethodPost,
                "/webhook",
                bytes.NewBuffer(body),
            )
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("X-API-Key", tt.apiKey)
            
            // 実行
            w := httptest.NewRecorder()
            app.Router.ServeHTTP(w, req)
            
            // 検証
            assert.Equal(t, tt.wantStatus, w.Code)
        })
    }
}
```

## 8. メイン関数

```go
// cmd/hokku/main.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "go.uber.org/zap"
    "hokku/internal/app"
)

func main() {
    // アプリケーション初期化
    application, err := app.New()
    if err != nil {
        panic(fmt.Sprintf("Failed to initialize app: %v", err))
    }
    defer application.Logger.Sync()
    
    // HTTPサーバー設定
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", application.Config.Server.Port),
        Handler:      application.Router,
        ReadTimeout:  application.Config.Server.ReadTimeout,
        WriteTimeout: application.Config.Server.WriteTimeout,
    }
    
    // Graceful shutdown設定
    go func() {
        application.Logger.Info("Server starting",
            zap.Int("port", application.Config.Server.Port),
        )
        
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            application.Logger.Fatal("Server failed to start", zap.Error(err))
        }
    }()
    
    // シグナル待機
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    application.Logger.Info("Shutting down server...")
    
    // タイムアウト付きシャットダウン
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        application.Logger.Error("Server forced to shutdown", zap.Error(err))
    }
    
    application.Logger.Info("Server exited")
}
```

## まとめ

この実用的な設計の利点：

1. **シンプルで理解しやすい**: 過度な抽象化を避け、Goらしい直接的な実装
2. **十分な品質**: 必要なエラーハンドリング、ロギング、テストを確保
3. **保守しやすい**: ファイル数が少なく、構造が明確
4. **拡張可能**: 必要に応じて機能追加が容易
5. **実装が速い**: ボイラープレートが少なく、すぐに動くものが作れる

Clean Architectureのような複雑な層構造は、大規模なプロジェクトや複数チームでの開発では有効ですが、Hokkuのようなシンプルなサービスには、この程度の構造で十分です。