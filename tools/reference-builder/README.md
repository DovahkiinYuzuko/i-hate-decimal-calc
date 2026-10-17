# reference-builder (`tools/reference-builder`)

ihd のリファレンスドキュメント（Markdown）から DADS 準拠の単一 HTML を生成するビルドツール / Build tool that generates DADS-compliant single-file HTML from ihd reference Markdown

[![Node.js Version](https://img.shields.io/badge/node-%3E%3D18.0.0-brightgreen.svg?style=flat-square&logo=node.js&logoColor=white)](https://nodejs.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](#日本語) | [English](#english)

## 日本語

`tools/reference-builder` は、`docs/references/{lang}/reference.{lang}.md` を入力ソースとして、デジタル庁デザインシステム（DADS）および GitHub Flavored Markdown（GFM）仕様に準拠した、外部リソース依存のない単一 HTML ファイル（Single-File HTML）を生成する独立ビルドツールです。

### 特徴とアーキテクチャ

1. **ASTパイプラインによる構文解析**:
   - `unified`、`remark-parse`、`remark-gfm`、`remark-math`、`remark-rehype`、`rehype-katex`、`rehype-stringify` による標準 AST 変換を行います。
2. **DADS × GFM の統合デザイン**:
   - GFM アラート（`[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, `[!CAUTION]`）を DADS の注釈ブロック・通知バナーとしてレンダリングします。
   - DADS テーブルスタイル（可変幅・レスポンシブ横スクロール保証）を適用します。
   - 見出し（H2 セクションおよび配下 H3 アコーディオン）に基づくアコーディオンおよび目次（TOC）を自動生成します。
3. **数式のサーバーサイド事前レンダリング (SSR)**:
   - `rehype-katex` により、ビルド時に数式を HTML+MathML へ事前展開します。
   - 生成された HTML 単体で表示可能であり、クライアントサイドでの KaTeX JavaScript 読み込みは不要です。
4. **外枠 UI 専用の i18n 辞書分離**:
   - 「本文へスキップ」「目次」「すべて展開」「検索プレースホルダー」などの外枠 UI テキストを `i18n/{lang}.json` に外出し管理します。
   - 英語 HTML（`reference.en.html`）への日本語テキスト混入を防止します。
5. **単一 HTML ファイルの生成 (Single-File HTML)**:
   - DADS CSS、viewer.js（検索・アコーディオン制御）、Base64 KaTeX WebFonts を 1 つの HTML ファイルへインラインバンドルします。
   - オフライン環境でも外部通信を行わずに閲覧・検索が可能です。

### ディレクトリ構成

```text
tools/reference-builder/
├── assets/
│   └── katex/
│       └── katex-inline.css    # Base64 WebFont埋め込み済み KaTeX CSS
├── i18n/              
│   └── {locales}.json          # 外枠UI辞書 (元となるMDファイルにないテキストを登録)
├── templates/
│   ├── layout.html             # HTML5骨格テンプレート
│   ├── dads.css                # DADSデザイントークン＋GFMスタイル定義
│   └── viewer.js               # 検索・アコーディオン・目次・コピー機能
├── index.js                    # unified ビルドエントリーポイント
├── package.json                # npm 依存関係
├── README.md                   # 本ドキュメント
├── NOTICE.md                   # サードパーティライセンス表示
└── LICENSE.MIT                 # MIT ライセンス
```

### ビルド手順

#### 依存関係のインストール

```bash
cd tools/reference-builder
npm ci
```

#### HTML リファレンスのビルド

リポジトリルートから以下のコマンドを実行します：

```bash
node tools/reference-builder/index.js
```

> [!NOTE]
> 正常に完了すると、HTMLファイルが更新されます：  
> `docs/references/{locale}/reference.{locale}.html`

CI（GitHub Actions `reference-sync.yml`）においても、Markdown 原稿と HTML 生成物の同期状態が自動検証されます。

### LICENSE
[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)

## English

`tools/reference-builder` is a standalone build tool that takes `docs/references/{lang}/reference.{lang}.md` as input and generates a self-contained Single-File HTML document adhering to Digital Agency Design System (DADS) and GitHub Flavored Markdown (GFM) specifications without external runtime dependencies.

### Features & Architecture

1. **Syntax Parsing via AST Pipeline**:
   - Performs standard AST transformations using `unified`, `remark-parse`, `remark-gfm`, `remark-math`, `remark-rehype`, `rehype-katex`, and `rehype-stringify`.
2. **Integrated DADS × GFM Design**:
   - Renders GFM alerts (`[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, `[!CAUTION]`) as DADS notification banners and callout blocks.
   - Applies DADS table styling with guaranteed responsive horizontal scrolling.
   - Automatically generates accordions and table of contents (TOC) based on headings (H2 sections and child H3 accordions).
3. **Server-Side Mathematical Typesetting (SSR)**:
   - Pre-renders mathematical equations into HTML+MathML at build time via `rehype-katex`.
   - The generated HTML displays independently without requiring client-side KaTeX JavaScript execution.
4. **Isolated i18n Dictionaries for UI Shell**:
   - Externalizes UI shell strings ("Skip to content", "Table of Contents", "Expand All", search placeholders, etc.) into `i18n/{lang}.json`.
   - Prevents unintended language mixing in localized editions.
5. **Single-File HTML Generation**:
   - Inlines DADS CSS, viewer.js (search, accordion controls), and Base64-encoded KaTeX WebFonts into a single HTML file.
   - Enables offline browsing and fast full-text searching without external network requests.

### Directory Structure

```text
tools/reference-builder/
├── assets/
│   └── katex/
│       └── katex-inline.css    # KaTeX CSS with Base64 embedded WebFonts
├── i18n/              
│   └── {locales}.json          # Shell UI dictionary (strings not present in source Markdown)
├── templates/
│   ├── layout.html             # HTML5 template skeleton
│   ├── dads.css                # DADS design tokens + GFM style definitions
│   └── viewer.js               # Search, accordion, TOC, and copy handlers
├── index.js                    # unified build entrypoint
├── package.json                # npm dependencies
├── README.md                   # This document
├── NOTICE.md                   # Third-party license notices
└── LICENSE.MIT                 # MIT License
```

### Build Instructions

#### Installing Dependencies

```bash
cd tools/reference-builder
npm ci
```

#### Building Reference HTML

Execute the following command from the repository root:

```bash
node tools/reference-builder/index.js
```

> [!NOTE]
> Upon successful completion, the HTML files are updated:  
> `docs/references/{locale}/reference.{locale}.html`

CI (GitHub Actions `reference-sync.yml`) automatically validates synchronization between Markdown sources and HTML build artifacts.

### LICENSE
[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)
