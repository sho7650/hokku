# Hokku - 高度設計仕様書 (Phase 3+の参考)

## ⚠️ 重要な警告

**このドキュメントは将来のPhase 3+で複雑性が必要になった場合の参考資料です。**
**現在のMVP優先アプローチとは矛盾するため、Phase 0-2では使用しないでください。**

**正しい実装順序:**
1. **Phase 0**: 50行main.go (IMPLEMENTATION_APPROACH.mdに従う)
2. **Phase 1-2**: 必要最小限の改善
3. **Phase 3**: 本当に必要になったらこの設計を検討

## Clean Architecture適用タイミング

### ❌ 早すぎる適用（やってはいけない）
```
Phase 0-2での以下は過剰設計:
├── domain/entities/           # エンティティ層
├── application/usecases/      # ユースケース層  
├── infrastructure/persistence/# 永続化層
└── interfaces/api/            # インターフェース層
```

### ✅ 適切な適用タイミング
```yaml
Clean Architecture適用条件:
  trigger_conditions:
    - 開発チーム: 5人以上
    - コード行数: 5000行以上
    - ビジネスロジック: 複雑化（10以上のユースケース）
    - データソース: 複数（DB + File + API等）
    - テスト: モック・スタブが困難
    
  before_applying:
    - Phase 0-2で実証済みの価値提供
    - 実際の複雑性による開発速度低下
    - チーム開発での責任分界点の必要性
```

## 段階的アーキテクチャ進化

### Phase 3A: 基本的な層分離 (必要性が明確になったら)
```
hokku/
├── cmd/hokku/main.go          # エントリーポイント
├── internal/
│   ├── handler/               # HTTPハンドラー
│   ├── service/               # ビジネスロジック
│   └── repository/            # データアクセス
└── pkg/                       # 共有ユーティリティ
```

### Phase 3B: 完全なClean Architecture (本当に必要な場合のみ)
```
hokku/
├── cmd/hokku/main.go          # エントリーポイント + DI設定
├── internal/
│   ├── domain/                # ビジネスドメイン層（依存性なし）
│   │   ├── entities/          
│   │   │   ├── webhook.go     # Webhookエンティティ
│   │   │   └── file.go        # Fileエンティティ
│   │   ├── repositories/      # リポジトリインターフェース
│   │   │   └── webhook_repository.go
│   │   └── services/          # ドメインサービス
│   │       └── webhook_service.go
│   ├── application/           # アプリケーション層
│   │   ├── usecases/          
│   │   │   ├── save_webhook.go # ユースケース実装
│   │   │   └── interfaces.go   # ユースケースインターフェース
│   │   └── dto/               # データ転送オブジェクト
│   │       ├── request.go     
│   │       └── response.go    
│   ├── infrastructure/        # インフラストラクチャ層
│   │   ├── persistence/       # 永続化実装
│   │   │   └── file_repository.go
│   │   ├── server/            # HTTPサーバー
│   │   │   └── router.go      # Ginルーター設定
│   │   └── config/            # 設定管理
│   │       └── config.go      
│   └── interfaces/            # インターフェース層
│       └── api/               # APIハンドラー
│           └── v1/            # バージョニング
│               └── webhook_handler.go
└── pkg/                       # 共有パッケージ（最小限）
```

## YAGNI適用による機能制限

### ❌ Phase 0-2で実装禁止項目
```yaml
過剰な抽象化:
  - エンティティ層の作成
  - リポジトリパターンの導入
  - ユースケース層の定義
  - 依存性注入フレームワーク
  - イベント駆動アーキテクチャ
  - CQRS/Event Sourcing
  - マイクロサービス分割

過剰な設定:
  - 設定ファイル（YAML/JSON）
  - 環境別設定管理
  - フィーチャーフラグ
  - 複数データベースサポート

過剰なミドルウェア:
  - 認証・認可システム
  - レート制限
  - キャッシング層
  - メッセージキュー
  - 監視・メトリクス
```

### ✅ 段階的導入指針
```yaml
Phase 0 (30分):
  implementation: "50行main.go"
  complexity: "最小限"
  
Phase 1 (1時間):
  implementation: "関数分離"
  complexity: "低"
  
Phase 2 (1時間): 
  implementation: "必要な抽象化のみ"
  complexity: "中"
  
Phase 3A (必要時):
  implementation: "基本的な層分離"
  complexity: "高"
  trigger: "チーム開発開始"
  
Phase 3B (本当に必要な場合):
  implementation: "Clean Architecture"
  complexity: "最高"
  trigger: "複雑なビジネス要件+大規模チーム"
```

## 適用判断フレームワーク

### Clean Architecture適用チェックリスト
```yaml
✅ 適用すべき状況:
  team_size: ">= 5人"
  codebase_size: ">= 5000行"
  business_complexity: "複数ドメイン"
  external_integrations: ">= 3個"
  testing_complexity: "モック必須レベル"
  
❌ 適用すべきでない状況:
  team_size: "< 3人"
  codebase_size: "< 1000行"  
  business_complexity: "単一ドメイン"
  external_integrations: "< 2個"
  development_speed: "最優先"
```

## まとめ

**⚠️ 警告**: このClean Architecture設計は高度な複雑性を前提としています。

**MVP原則**: 
- Phase 0-2では `IMPLEMENTATION_APPROACH.md` に従う
- この設計はPhase 3+で本当に必要になった場合のみ参考にする
- 「将来必要になるかも」は実装しない理由

**判断基準**: 現在の実装で**実際の問題**が発生してから適用を検討してください。