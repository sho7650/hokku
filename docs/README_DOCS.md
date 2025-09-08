# 📚 Hokku ドキュメント一覧

## 🎯 MVP優先実装ドキュメント（Phase 0-2で使用）

### 主要ドキュメント（実装時に必ず参照）

| ドキュメント | 役割 | 使用タイミング |
|------------|------|--------------|
| **[IMPLEMENTATION_APPROACH.md](IMPLEMENTATION_APPROACH.md)** | MVP優先実装戦略の基本方針 | **Phase 0-2全体** |
| **[IMPLEMENTATION_WORKFLOW.md](IMPLEMENTATION_WORKFLOW.md)** | 段階的実装ワークフロー | **実装作業時** |
| **[PROJECT_EXECUTION_PLAN.md](PROJECT_EXECUTION_PLAN.md)** | 実行計画とタイムライン | **プロジェクト管理** |

### 設計指針ドキュメント

| ドキュメント | 役割 | 使用タイミング |
|------------|------|--------------|
| **[SOLID_YAGNI_COMPLIANT_DESIGN.md](SOLID_YAGNI_COMPLIANT_DESIGN.md)** | 段階的SOLID適用戦略 | **設計判断時** |

---

## 📋 Phase 3+参考ドキュメント（将来の拡張時のみ）

### ⚠️ 注意: Phase 0-2では使用禁止

| ドキュメント | 内容 | 適用条件 |
|------------|------|---------|
| **[PRACTICAL_DESIGN.md](PRACTICAL_DESIGN.md)** | 実用的な構造設計 | コード300行超過時 |
| **[IMPROVED_DESIGN.md](IMPROVED_DESIGN.md)** | Clean Architecture設計 | チーム5人以上、コード5000行以上 |
| **[QUALITY_GATES_FRAMEWORK.md](QUALITY_GATES_FRAMEWORK.md)** | 品質ゲートフレームワーク | 本格的な品質管理が必要時 |

---

## 🚀 実装の進め方

### Phase 0: 最小動作版（30分）
```bash
# 1. IMPLEMENTATION_APPROACH.md のPhase 0コードをコピー
# 2. main.go作成（50行、Google Go Style Guide準拠）
# 3. 実行: go run main.go
# 4. テスト: curl -X POST localhost:8080/webhook -d '{"test":"data"}'
```

### Phase 1: 基本品質向上（1時間）
- titleチェック追加
- storageディレクトリ作成
- /health エンドポイント
- **参照**: IMPLEMENTATION_WORKFLOW.md の Phase 1

### Phase 2: セキュリティ最小限（1時間）  
- ファイル名サニタイゼーション
- パストラバーサル防止
- 環境変数設定
- **参照**: IMPLEMENTATION_WORKFLOW.md の Phase 2

---

## 🔑 重要原則

### MVP優先・YAGNI徹底
```yaml
実装判断:
  今必要: → 実装する ✅
  将来必要かも: → 実装しない ❌
```

### 段階的複雑性管理
```yaml
Phase 0: 10-50行の単一ファイル
Phase 1: 100-200行、基本的な関数分離
Phase 2: 300-500行、必要な抽象化のみ
Phase 3: 本当に必要になったら複雑設計を検討
```

---

## 📝 ドキュメント管理方針

- **Phase 0-2**: 主要3ドキュメントのみ参照
- **Phase 3+**: 実際の問題が発生してから参考ドキュメントを確認
- **原則**: 「将来必要になるかも」は読まない理由