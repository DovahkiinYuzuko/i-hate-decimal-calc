# Reference Builder (`tools/reference-builder`)

`tools/reference-builder` は、`docs/references/{lang}/reference.{lang}.md` を入力ソースとして、デジタル庁デザインシステム（DADS）準拠かつ GitHub Flavored Markdown（GFM）仕様に準拠した、外部リソース依存のない単一 HTML ファイル（Single-File HTML）を生成する独立ビルドツールです。

---

## 1. 特徴とアーキテクチャ

1. **ASTパイプラインによる言語非依存の構文解析**:
   - `unified` + `remark-parse` + `remark-gfm` + `remark-math` + `remark-rehype` + `rehype-katex` + `rehype-stringify` による標準AST（mdast / hast）変換。
   - 手書きの正規表現パースや Unicode 退避ハックを排除。
2. **DADS × GFM の統合デザイン**:
   - GFM アラート（`[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, `[!CAUTION]`）を DADS の注釈ブロック・通知バナーとしてレンダリング。
   - DADS テーブルスタイル（可変幅・レスポンシブ横スクロール保証）。
   - 見出し（H2セクション ＋ 配下H3アコーディオン）に基づく機械的なアコーディオン・目次（TOC）自動生成。
3. **数式のサーバーサイド事前レンダリング (SSR)**:
   - `rehype-katex` により、ビルド時に数式を HTML+MathML へ事前展開。
   - 生成された HTML 単体で表示可能とし、クライアントサイドでの KaTeX JavaScript 読み込みを不要化。
4. **外枠 UI 専用の i18n 辞書分離**:
   - 「本文へスキップ」「目次」「すべて展開」「検索プレースホルダー」などの外枠UIテキストを `i18n/{lang}.json` に外出し管理。
   - 英語HTML（`reference.en.html`）への日本語テキスト混入を防止。
5. **単一 HTML ファイルの生成 (Single-File HTML)**:
   - DADS CSS、viewer.js（検索・アコーディオン制御）、Base64 KaTeX WebFonts をビルド時に1つの HTML ファイルへインラインバンドル。
   - オフライン環境でも外部通信を行わずに閲覧・検索可能。

---

## 2. ディレクトリ構成

```text
tools/reference-builder/
├── assets/
│   └── katex/
│       └── katex-inline.css    # Base64 WebFont埋め込み済み KaTeX CSS
├── i18n/              
│   └── {locales}.json          # 外枠UI辞書(元となるMDファイルにないテキストを登録)
├── templates/
│   ├── layout.html             # HTML5骨格テンプレート
│   ├── dads.css                # DADSデザイントークン＋GFMスタイル定義
│   └── viewer.js               # 検索・アコーディオン・目次・コピー機能
├── index.js                    # unified ビルドエントリーポイント
├── package.json                # npm 依存関係
├── README.md                   # 本ドキュメント
└── NOTICE.md                   # サードパーティライセンス表示
```

---

## 3. ビルド手順

### 依存関係のインストール

```bash
cd tools/reference-builder
npm ci
```

### HTML リファレンスのビルド

リポジトリルートから以下のコマンドを実行します：

```bash
node tools/reference-builder/index.js
```

> [!NOTE]
> 正常に完了すると、HTMLファイルが更新されます：  
> `docs/references/{locale}/reference.{locale}.html`


CI（GitHub Actions `reference-sync.yml`）においても、Markdown 原稿と HTML 生成物の同期状態が自動検証されます。
