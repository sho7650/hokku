# Hokku Implementation Approach - MVP優先実装戦略

## 実装の基本原則

### ❌ 間違ったアプローチ（過剰設計）
```
1. インターフェース定義から開始
2. 抽象化層の構築
3. 依存性注入の実装
4. やっとHTTPハンドラ実装
5. ようやく動作確認
```

### ✅ 正しいアプローチ（MVP優先）
```
1. 動作する最小コードを書く（50行程度）
2. 実際に動かして確認
3. 必要に応じて機能追加
4. リファクタリング（必要な場合のみ）
5. 抽象化（本当に必要になったら）
```

## Phase別実装計画（修正版）

### Phase 0: 最小動作版（30分）
**目標**: とにかく動くものを作る

```go
// main.go - 完全に動作する最小版
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
)

func main() {
    // webhookエンドポイント (Google Go Style Guide準拠: 関数分離)
    http.HandleFunc("/webhook", handleWebhook)
    
    // サーバー起動
    fmt.Println("Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}

// handleWebhook はWebhookリクエストを処理する (Google準拠: 単一責任)
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // リクエストボディを読む (Google準拠: エラーハンドリング)
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Cannot read body", 400)
        return
    }
    
    // JSONかどうか確認 (Google準拠: エラーハンドリング)
    var data map[string]interface{}
    if err := json.Unmarshal(body, &data); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }
    
    // ファイルに保存 (Google準拠: エラーハンドリング)
    filename := fmt.Sprintf("webhook_%d.json", time.Now().Unix())
    if err := os.WriteFile(filename, body, 0644); err != nil {
        http.Error(w, "Cannot save file", 500)
        return
    }
    
    // 成功レスポンス
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "saved",
        "file":   filename,
    })
}
```

**動作確認**:
```bash
go run main.go
curl -X POST localhost:8080/webhook -d '{"test":"data"}'
```

### Phase 1: 基本的な改善（1時間）
**目標**: 最低限の品質向上

```go
// 追加する機能のみ
+ storageディレクトリ作成
+ タイトルフィールド必須化
+ 基本的なログ出力
+ ヘルスチェックエンドポイント
```

**実装差分**:
```go
// ディレクトリ作成
os.MkdirAll("storage", 0755)

// タイトルチェック
if data["title"] == nil || data["title"] == "" {
    http.Error(w, "title is required", 400)
    return  
}

// ファイル名にタイトルを含める
title := fmt.Sprintf("%v", data["title"])
filename := fmt.Sprintf("storage/%d_%s.json", time.Now().Unix(), title)
```

### Phase 2: セキュリティ最小限（1時間）
**目標**: 最低限のセキュリティ

```go
// 追加する機能のみ
+ ファイル名サニタイゼーション（危険文字を_に置換）
+ ../ を含むパスの拒否
+ 環境変数での設定（PORT、STORAGE_PATH）
```

**実装例**:
```go
// 危険な文字を置換
func sanitize(s string) string {
    // 最小限のサニタイゼーション
    s = strings.ReplaceAll(s, "/", "_")
    s = strings.ReplaceAll(s, "..", "_")
    s = strings.ReplaceAll(s, "\x00", "_")
    return s
}
```

### Phase 3: 設定管理（必要になったら）
**目標**: 設定の外部化

```go
// 環境変数から読み込み
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

storagePath := os.Getenv("STORAGE_PATH")
if storagePath == "" {
    storagePath = "./storage"
}
```

### Phase 4: エラーハンドリング改善（必要になったら）
**目標**: より良いエラーメッセージ

```go
// JSONエラーレスポンス
type ErrorResponse struct {
    Error string `json:"error"`
    Code  int    `json:"code"`
}

func sendError(w http.ResponseWriter, message string, code int) {
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: message,
        Code:  code,
    })
}
```

### Phase 5: リファクタリング（必要性が明確になったら）
**目標**: コードの整理

- ハンドラを関数に分離
- 設定を構造体にまとめる
- ファイル操作をヘルパー関数に

**重要**: この段階でもインターフェースは不要！

### Phase 6: 抽象化（本当に必要になったら）
**目標**: テスタビリティやモジュール性が必要になったら

- インターフェース定義
- 依存性注入
- モック可能な設計

## YAGNI原則の適用

### ❌ 最初から実装してはいけないもの
- インターフェース定義
- 抽象化層
- 複雑な検証ロジック
- プラガブルなストレージバックエンド
- メトリクス収集
- 分散トレーシング
- イベントバス
- プラグインシステム

### ✅ 必要になってから追加するもの
- 認証（外部公開時）
- レート制限（負荷問題発生時）
- データベース保存（ファイルで問題が出たら）
- 非同期処理（パフォーマンス問題時）
- キャッシング（必要性が明確になったら）

## 判断基準

### 機能追加の判断フロー
```
1. 現在の実装で問題があるか？
   No → 追加しない
   Yes → 2へ

2. 問題は頻繁に発生するか？
   No → 追加しない
   Yes → 3へ

3. シンプルな解決策はあるか？
   Yes → シンプルな解決策を実装
   No → 4へ

4. 複雑な解決策の価値は明確か？
   No → 追加しない
   Yes → 最小限の実装
```

## アンチパターンの回避

### 1. 早すぎる抽象化
```go
// ❌ 悪い例：最初からインターフェース
type Storage interface {
    Save(data []byte) error
}

type FileStorage struct{}
type S3Storage struct{}  // 使わないのに定義

// ✅ 良い例：具体的な実装のみ
func saveToFile(data []byte, filename string) error {
    return os.WriteFile(filename, data, 0644)
}
```

### 2. 過剰な設定可能性
```go
// ❌ 悪い例：使わない設定項目
type Config struct {
    Port                int
    StoragePath         string
    MaxFileSize         int64
    MinFileSize         int64
    AllowedExtensions   []string
    ForbiddenExtensions []string
    RetryCount          int
    RetryDelay          time.Duration
    // ... 50個の設定項目
}

// ✅ 良い例：必要最小限
port := "8080"
storagePath := "./storage"
```

### 3. 将来の拡張性への過剰な配慮
```go
// ❌ 悪い例：プラグインシステム
type Plugin interface {
    Process(data []byte) ([]byte, error)
}

type PluginManager struct {
    plugins []Plugin
}

// ✅ 良い例：直接実装
func processWebhook(data []byte) error {
    // 直接処理
    return saveToFile(data, "webhook.json")
}
```

## 実装の進化例

### Step 1: 最小実装（10行）
```go
http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    os.WriteFile("webhook.json", body, 0644)
    w.WriteHeader(200)
})
```

### Step 2: エラーハンドリング追加（+5行）
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

### Step 3: JSONバリデーション追加（+5行）
```go
var data map[string]interface{}
if err := json.Unmarshal(body, &data); err != nil {
    http.Error(w, "Invalid JSON", 400)
    return
}
```

### Step 4: 必要になったらリファクタリング
- ハンドラを別関数に
- 設定を外部化
- テストを追加

## まとめ

1. **動くコードを最優先**
2. **必要性が明確になってから追加**
3. **シンプルさを維持**
4. **抽象化は最後の手段**
5. **YAGNIを徹底**

「将来必要になるかも」は実装しない理由。
「今必要」だけが実装する理由。