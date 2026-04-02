# mdview

Markdownファイルをターミナル上で美しくレンダリングするGo製CLIツール。

ANSIエスケープシーケンス、Unicodeボックス描画文字、Sixelグラフィックスを活用し、ターミナル環境に最適化された出力を生成します。

## 機能

- **見出し** — レベル別の色・太字・下線で階層構造を視覚化
- **テキスト装飾** — 太字、斜体、取り消し線、インラインコード
- **コードブロック** — ボックス描画文字の枠、行番号、言語ラベル、背景色
- **リスト** — 順序付き/順序なし、ネストレベル別Unicode記号（•◦▪）、桁揃え
- **テーブル** — ボックス描画罫線、ヘッダー装飾、列幅自動調整、アライメント
- **引用ブロック** — ネストレベル別の色付き縦線、斜体テキスト
- **画像表示** — Sixel対応ターミナルで画像をインライン表示（PNG/JPEG/GIF）
- **Mermaid図** — mmdcによるPNG変換 → Sixel表示
- **ページャー** — less風のスクロール、検索、ステータスバー
- **Pretty Printer** — ASTからMarkdownを再生成（ラウンドトリップ保証）
- **環境適応** — TrueColor/256色の自動検出、NO_COLOR対応、パイプ出力時のプレーンテキスト化

## インストール

```bash
go install github.com/user/mdrender/cmd/mdview@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/user/mdview.git
cd mdview
go build -o build/mdview ./cmd/mdview/
```

## 使い方

```bash
# ファイルを指定
mdview README.md

# パイプ入力
cat README.md | mdview

# Pretty Printモード（Markdown再生成）
mdview --pretty-print README.md

# Mermaidテーマ指定
mdview --mermaid-theme dark README.md

# ページャー無効
mdview --no-pager README.md

# 色無効
mdview --no-color README.md
```

## オプション

| オプション | 説明 |
|---|---|
| `--mermaid-theme <theme>` | Mermaidテーマ（default / dark / forest / neutral） |
| `--pretty-print` | ASTからMarkdownを再生成して出力 |
| `--no-pager` | ページャーモードを無効化 |
| `--no-color` | 色・装飾を無効化してプレーンテキスト出力 |

## ページャー操作

| キー | 操作 |
|---|---|
| `j` / `↓` | 1行下スクロール |
| `k` / `↑` | 1行上スクロール |
| `Space` / `PgDn` | 1ページ下 |
| `b` / `PgUp` | 1ページ上 |
| `g` | 先頭へ |
| `G` | 末尾へ |
| `/` | 検索 |
| `n` | 次の検索結果 |
| `N` | 前の検索結果 |
| `q` | 終了 |

## 依存ライブラリ

- [goldmark](https://github.com/yuin/goldmark) — CommonMark準拠Markdownパーサー
- [rapid](https://pgregory.net/rapid) — プロパティベーステスト
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — ターミナル制御

## Mermaid図の表示

Mermaid図の表示には [mermaid-cli](https://github.com/mermaid-js/mermaid-cli) が必要です:

```bash
npm install -g @mermaid-js/mermaid-cli
```

## テスト

```bash
go test ./...
```

29個のプロパティベーステストを含む包括的なテストスイートで正確性を検証しています。

## ライセンス

MIT
