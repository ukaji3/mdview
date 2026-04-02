# mdview

Markdownファイルをターミナル上で美しくレンダリングするGo製CLIツール。

ANSIエスケープシーケンス、Unicodeボックス描画文字、複数の画像プロトコル（Kitty / iTerm2 / Sixel）を活用し、ターミナル環境に最適化された出力を生成します。

## 機能

- **見出し** — レベル別の色・太字・下線で階層構造を視覚化
- **テキスト装飾** — 太字、斜体、取り消し線、インラインコード
- **コードブロック** — ボックス描画文字の枠、行番号、言語ラベル、背景色
- **リスト** — 順序付き/順序なし、ネストレベル別Unicode記号（•◦▪）、桁揃え
- **テーブル** — ボックス描画罫線、ヘッダー装飾、列幅自動調整、アライメント
- **引用ブロック** — ネストレベル別の色付き縦線、斜体テキスト
- **画像表示** — Kitty / iTerm2 / Sixel プロトコルを自動検出してインライン表示（PNG/JPEG/GIF）
- **Mermaid図** — mmdcによるPNG変換 → ターミナル内画像表示
- **ページャー** — less風のスクロール、検索、ステータスバー
- **ターミナルリサイズ対応** — ウィンドウサイズ変更時に自動再描画
- **ファイル監視** — 表示中のMarkdownファイルの変更を検知して自動再描画
- **Pretty Printer** — ASTからMarkdownを再生成（ラウンドトリップ保証）
- **環境適応** — TrueColor/256色の自動検出、NO_COLOR対応、パイプ出力時のプレーンテキスト化

## 対応ターミナル（画像表示）

| ターミナル | プロトコル | 自動検出 |
|---|---|---|
| Kitty | Kitty Graphics Protocol | ✓ |
| Ghostty | Kitty Graphics Protocol | ✓ |
| WezTerm | iTerm2 Inline Images | ✓ |
| iTerm2 | iTerm2 Inline Images | ✓ |
| mintty | iTerm2 Inline Images | ✓ |
| mlterm, foot 等 | Sixel | ✓ |

`mdview --check-image` で対応状況を確認できます。

## インストール

```bash
go install github.com/ukaji3/mdview/cmd/mdview@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/ukaji3/mdview.git
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

# 画像プロトコル対応チェック
mdview --check-image
```

## オプション

| オプション | 説明 |
|---|---|
| `--mermaid-theme <theme>` | Mermaidテーマ（default / dark / forest / neutral） |
| `--pretty-print` | ASTからMarkdownを再生成して出力 |
| `--no-pager` | ページャーモードを無効化 |
| `--no-color` | 色・装飾を無効化してプレーンテキスト出力 |
| `--check-image` | ターミナルの画像プロトコル対応状況を表示 |

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

ページャー表示中にターミナルをリサイズすると自動的に再描画されます。ファイルを指定して表示している場合、ファイルの変更も自動検知して再描画します。いずれの場合もスクロール位置は保持されます。

## 依存ライブラリ

- [goldmark](https://github.com/yuin/goldmark) — CommonMark準拠Markdownパーサー
- [go-runewidth](https://github.com/mattn/go-runewidth) — Unicode文字幅計算
- [rapid](https://pgregory.net/rapid) — プロパティベーステスト
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — ターミナル制御
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) — システムコール（TIOCGWINSZ）

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
