#!/usr/bin/env node

/**
 * tools/reference-builder/index.js
 * 
 * docs/references/{lang}/reference.{lang}.md を原典（SSOT）として読み込み、
 * DADS（デジタル庁デザインシステム）準拠 & GFM拡張対応の完全自己完結Single-File HTMLを生成する。
 * 
 * - unified + remark-parse + remark-gfm + remark-math + rehype-katex によるAST構文木処理
 * - 数式のサーバーサイド事前レンダリング (SSR) によりブラウザ側KaTeX JSを完全撤廃 (0KB化)
 * - GFM アラート ([!NOTE], [!TIP], [!IMPORTANT], [!WARNING], [!CAUTION]) のDADS注釈ブロック化
 * - 純粋な見出し階層に基づく言語非依存のアコーディオン・目次生成
 * - 外枠UI専用の i18n 辞書 (i18n/{lang}.json) の完全分離
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import remarkRehype from 'remark-rehype';
import rehypeKatex from 'rehype-katex';
import rehypeStringify from 'rehype-stringify';
import { visit } from 'unist-util-visit';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ROOT_DIR = path.resolve(__dirname, '../..');
const REFERENCES_DIR = path.join(ROOT_DIR, 'docs', 'references');
const TEMPLATES_DIR = path.join(__dirname, 'templates');
const ASSETS_DIR = path.join(__dirname, 'assets');
const I18N_DIR = path.join(__dirname, 'i18n');

/**
 * HTML エスケープ処理
 */
function escapeHtml(str) {
  if (!str) return '';
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/**
 * 見出しテキストからアンカー ID を生成
 */
function slugify(text) {
  return 'sec-' + text.toLowerCase()
    .replace(/[^a-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * AST ノードからプレーンテキストを抽出
 */
function toString(node) {
  if (!node) return '';
  if (node.value) return node.value;
  if (node.children) return node.children.map(toString).join('');
  return '';
}

/**
 * GFM アラート ([!NOTE] 等) を DADS 注釈ブロック / ノティフィケーションバナーへ変換する remark プラグイン
 */
function remarkGfmAlerts() {
  const alertIcons = {
    note: '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>',
    tip: '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>',
    important: '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>',
    warning: '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/></svg>',
    caution: '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>'
  };

  const alertTitles = {
    note: 'Note',
    tip: 'Tip',
    important: 'Important',
    warning: 'Warning',
    caution: 'Caution'
  };

  return (tree) => {
    visit(tree, 'blockquote', (node) => {
      if (!node.children || node.children.length === 0) return;
      const firstChild = node.children[0];
      if (firstChild.type !== 'paragraph' || !firstChild.children || firstChild.children.length === 0) return;
      const firstTextNode = firstChild.children[0];
      if (firstTextNode.type !== 'text') return;

      const match = firstTextNode.value.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?:\r?\n)?/i);
      if (!match) return;

      const alertType = match[1].toLowerCase();
      firstTextNode.value = firstTextNode.value.slice(match[0].length);

      node.data = node.data || {};
      node.data.hName = 'div';
      node.data.hProperties = {
        className: ['markdown-alert', `markdown-alert-${alertType}`]
      };

      const titleHtml = `${alertIcons[alertType] || ''}<span>${alertTitles[alertType]}</span>`;
      const titleNode = {
        type: 'html',
        value: `<div class="markdown-alert-title">${titleHtml}</div>`
      };

      node.children.unshift(titleNode);
    });
  };
}

/**
 * テーブル・インラインコード・コードブロックの DADS 装飾を行う rehype プラグイン
 */
function rehypeDadsEnhance(copyTooltipText) {
  return (tree) => {
    // 1. テーブルの DADS スタイル化 ＋ 横スクロールコンテナ化
    visit(tree, 'element', (node, index, parent) => {
      if (node.tagName === 'table') {
        node.properties = node.properties || {};
        node.properties.className = ['dads-table'];

        // 親が既に table-container でなければラップ
        if (parent && parent.tagName !== 'div' && (!parent.properties || !parent.properties.className || !parent.properties.className.includes('table-container'))) {
          const wrapper = {
            type: 'element',
            tagName: 'div',
            properties: { className: ['table-container'] },
            children: [node]
          };
          parent.children[index] = wrapper;
        }
      }

      // 2. インラインコードのクリックコピー化
      if (node.tagName === 'code' && parent && parent.tagName !== 'pre') {
        node.properties = node.properties || {};
        const classes = Array.isArray(node.properties.className) ? node.properties.className : [];
        if (!classes.includes('clickable-code')) {
          classes.push('clickable-code');
          node.properties.className = classes;
          const text = toString(node).trim();
          node.properties.dataCopy = text;
          node.properties.title = copyTooltipText;
        }
      }

      // 3. コードブロック (pre > code) の DADS REPL カード化
      if (node.tagName === 'pre' && node.children && node.children[0] && node.children[0].tagName === 'code') {
        const codeNode = node.children[0];
        const codeClasses = Array.isArray(codeNode.properties?.className) ? codeNode.properties.className : [];
        const langClass = codeClasses.find(c => typeof c === 'string' && c.startsWith('language-'));
        const langName = langClass ? langClass.replace('language-', '').toUpperCase() : 'CONSOLE';

        node.properties = node.properties || {};
        node.properties.className = ['dads-code-pre'];

        if (parent && (!parent.properties || !parent.properties.className || !parent.properties.className.includes('code-block-wrapper'))) {
          const wrapper = {
            type: 'element',
            tagName: 'div',
            properties: { className: ['code-block-wrapper'] },
            children: [
              {
                type: 'element',
                tagName: 'div',
                properties: { className: ['code-block-header'] },
                children: [
                  {
                    type: 'element',
                    tagName: 'span',
                    properties: { className: ['code-block-title'] },
                    children: [{ type: 'text', value: langName }]
                  },
                  {
                    type: 'element',
                    tagName: 'button',
                    properties: {
                      type: 'button',
                      className: ['btn-copy-code'],
                      'aria-label': 'Copy code'
                    },
                    children: [
                      {
                        type: 'raw',
                        value: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>'
                      }
                    ]
                  }
                ]
              },
              node
            ]
          };
          parent.children[index] = wrapper;
        }
      }
    });
  };
}

/**
 * 単一 Markdown ファイルのビルド処理
 */
async function buildLanguageEdition(lang) {
  const langDir = path.join(REFERENCES_DIR, lang);
  const mdFile = path.join(langDir, `reference.${lang}.md`);
  const htmlFile = path.join(langDir, `reference.${lang}.html`);

  if (!fs.existsSync(mdFile)) {
    console.warn(`[Build Warning] Markdown file not found: ${mdFile}, skipping...`);
    return;
  }

  console.log(`[Build] Processing ${lang} edition: ${mdFile}`);

  // i18n 辞書の読み込み
  const i18nFile = path.join(I18N_DIR, `${lang}.json`);
  const fallbackI18nFile = path.join(I18N_DIR, 'en.json');
  const dictPath = fs.existsSync(i18nFile) ? i18nFile : fallbackI18nFile;
  const i18n = JSON.parse(fs.readFileSync(dictPath, 'utf-8'));

  const mdContent = fs.readFileSync(mdFile, 'utf-8');

  // Markdown を mdast へパース
  const parser = unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkMath)
    .use(remarkGfmAlerts);

  const mdast = parser.parse(mdContent);
  await parser.run(mdast);

  // AST からタイトル (H1)、リード文、および各セクションを抽出
  let title = 'ihd Complete Reference';
  let leadText = '';
  let inLead = false;

  const sections = []; // 各要素: { id, title, nodes: [], keywords: [], isGroupHeader: boolean, subSections: [] }
  let currentH2Section = null;
  let currentH3Section = null;

  // AST トップレベルノードの走査・分割
  const nodes = mdast.children;
  let idx = 0;

  while (idx < nodes.length) {
    const node = nodes[idx];

    // 1. H1 タイトル
    if (node.type === 'heading' && node.depth === 1) {
      title = toString(node).trim();
      inLead = true;
      idx++;
      continue;
    }

    // 2. H2 大見出し
    if (node.type === 'heading' && node.depth === 2) {
      inLead = false;
      const secTitle = toString(node).trim();
      const secId = slugify(secTitle);

      // 原典に手動で書かれた目次ブロック（目次 / Table of Contents / 目录）はスキップ
      if (/^目次|^Table of Contents|^目錄|^目录/i.test(secTitle)) {
        idx++;
        // 次の H2 が来るまでスキップ
        while (idx < nodes.length && !(nodes[idx].type === 'heading' && nodes[idx].depth <= 2)) {
          idx++;
        }
        continue;
      }

      currentH2Section = {
        id: secId,
        title: secTitle,
        nodes: [],
        keywords: [secTitle],
        subSections: []
      };
      sections.push(currentH2Section);
      currentH3Section = null;
      idx++;
      continue;
    }

    // 3. H3 中見出し
    if (node.type === 'heading' && node.depth === 3) {
      inLead = false;
      const subTitle = toString(node).trim();
      const subId = slugify(subTitle);

      if (currentH2Section) {
        currentH3Section = {
          id: subId,
          title: subTitle,
          nodes: [],
          keywords: [subTitle]
        };
        currentH2Section.subSections.push(currentH3Section);
      }
      idx++;
      continue;
    }

    // 4. リード文の収集
    if (inLead && node.type === 'paragraph' && !leadText) {
      leadText = toString(node).trim();
      idx++;
      continue;
    }

    // 5. 本文ノードの格納
    if (currentH3Section) {
      currentH3Section.nodes.push(node);
      const text = toString(node);
      if (text) currentH3Section.keywords.push(...text.split(/\s+/).slice(0, 10));
    } else if (currentH2Section) {
      currentH2Section.nodes.push(node);
      const text = toString(node);
      if (text) currentH2Section.keywords.push(...text.split(/\s+/).slice(0, 10));
    }

    idx++;
  }

  // HTML 変換プロセッサの初期化
  const htmlProcessor = unified()
    .use(remarkRehype, { allowDangerousHtml: true })
    .use(rehypeKatex, { output: 'htmlAndMathml', throwOnError: false })
    .use(rehypeDadsEnhance, i18n.copyTooltip)
    .use(rehypeStringify, { allowDangerousHtml: true });

  // 各セクションのノードを HTML に変換
  for (const sec of sections) {
    if (sec.nodes.length > 0) {
      const rootNode = { type: 'root', children: sec.nodes };
      const result = await htmlProcessor.run(rootNode);
      sec.contentHtml = htmlProcessor.stringify(result);
    } else {
      sec.contentHtml = '';
    }

    for (const sub of sec.subSections) {
      const rootNode = { type: 'root', children: sub.nodes };
      const result = await htmlProcessor.run(rootNode);
      sub.contentHtml = htmlProcessor.stringify(result);
    }
  }

  // 目次 (TOC HTML) の生成
  let tocHtml = '<ul class="toc-list">';
  sections.forEach(sec => {
    if (sec.subSections.length > 0) {
      const subAria = i18n.subAccordionToggleAria.replace('{title}', escapeHtml(sec.title));
      tocHtml += `
        <li class="toc-item">
          <details class="toc-group" open>
            <summary class="toc-group-summary" aria-label="${subAria}">
              <span>${escapeHtml(sec.title)}</span>
              <svg class="toc-group-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/></svg>
            </summary>
            <ul class="toc-sublist">
      `;
      sec.subSections.forEach(sub => {
        tocHtml += `<li class="toc-subitem"><a class="toc-sublink" href="#${sub.id}">${escapeHtml(sub.title)}</a></li>`;
      });
      tocHtml += `
            </ul>
          </details>
        </li>
      `;
    } else {
      tocHtml += `<li class="toc-item"><a class="toc-link" href="#${sec.id}">${escapeHtml(sec.title)}</a></li>`;
    }
  });
  tocHtml += '</ul>';

  // 本文セクションアコーディオン HTML の生成
  let sectionsHtml = '';
  let accordionCount = 0;

  sections.forEach(sec => {
    if (sec.subSections.length > 0) {
      // H3 サブセクションを持つグループ（例: 関数カテゴリ一覧）
      sectionsHtml += `<h2 class="dads-h2" id="${sec.id}">${escapeHtml(sec.title)}</h2>`;
      if (sec.contentHtml) {
        sectionsHtml += `<div class="group-intro">${sec.contentHtml}</div>`;
      }
      sec.subSections.forEach(sub => {
        const isOpen = accordionCount < 4 ? ' open' : '';
        const ariaLabel = i18n.accordionToggleAria.replace('{title}', escapeHtml(sub.title));
        sectionsHtml += `
          <details class="dads-accordion" id="${sub.id}"${isOpen} data-keywords="${escapeHtml(sub.keywords.slice(0, 50).join(' '))}">
            <summary class="accordion-summary" aria-label="${ariaLabel}">
              <span class="accordion-title">${escapeHtml(sub.title)}</span>
              <svg class="accordion-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg>
            </summary>
            <div class="accordion-content">
              ${sub.contentHtml}
            </div>
          </details>
        `;
        accordionCount++;
      });
    } else {
      // 通常の H2 セクション（演算子、リテラル、予約定数など）
      const isOpen = accordionCount < 4 ? ' open' : '';
      const ariaLabel = i18n.accordionToggleAria.replace('{title}', escapeHtml(sec.title));
      sectionsHtml += `
        <details class="dads-accordion" id="${sec.id}"${isOpen} data-keywords="${escapeHtml(sec.keywords.slice(0, 50).join(' '))}">
          <summary class="accordion-summary" aria-label="${ariaLabel}">
            <span class="accordion-title">${escapeHtml(sec.title)}</span>
            <svg class="accordion-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg>
          </summary>
          <div class="accordion-content">
            ${sec.contentHtml}
          </div>
        </details>
      `;
      accordionCount++;
    }
  });

  // テンプレートおよび静的アセットの読み込み
  const layoutTemplate = fs.readFileSync(path.join(TEMPLATES_DIR, 'layout.html'), 'utf-8');
  const dadsCss = fs.readFileSync(path.join(TEMPLATES_DIR, 'dads.css'), 'utf-8');
  const viewerJs = fs.readFileSync(path.join(TEMPLATES_DIR, 'viewer.js'), 'utf-8');
  const katexCss = fs.readFileSync(path.join(ASSETS_DIR, 'katex', 'katex-inline.css'), 'utf-8');

  // プレースホルダーのインライン置換
  let html = layoutTemplate
    .replace('lang="en" data-lang="{{LANG}}"', `lang="${lang}"`)
    .replace(/\{\{LANG\}\}/g, lang)
    .replace(/\{\{META_DESCRIPTION\}\}/g, escapeHtml(leadText || title))
    .replace(/\{\{TITLE\}\}/g, escapeHtml(title))
    .replace(/\/\*\s*\{\{KATEX_CSS\}\}\s*\*\//g, katexCss)
    .replace(/\{\{KATEX_CSS\}\}/g, katexCss)
    .replace(/\/\*\s*\{\{DADS_CSS\}\}\s*\*\//g, dadsCss)
    .replace(/\{\{DADS_CSS\}\}/g, dadsCss)
    .replace(/\{\{SKIP_LINK\}\}/g, escapeHtml(i18n.skipLink))
    .replace(/\{\{HAMBURGER_ARIA\}\}/g, escapeHtml(i18n.hamburgerAria))
    .replace(/\{\{DRAWER_TITLE\}\}/g, escapeHtml(i18n.drawerTitle))
    .replace(/\{\{DRAWER_CLOSE_ARIA\}\}/g, escapeHtml(i18n.drawerCloseAria))
    .replace(/\{\{TOC_TITLE\}\}/g, escapeHtml(i18n.tocTitle))
    .replace(/\{\{TOC_ARIA\}\}/g, escapeHtml(i18n.tocAria))
    .replace(/\{\{TOC_HTML\}\}/g, tocHtml)
    .replace(/\{\{LEAD_HTML\}\}/g, leadText ? `<p class="lead-text">${escapeHtml(leadText)}</p>` : '')
    .replace(/\{\{SEARCH_PLACEHOLDER\}\}/g, escapeHtml(i18n.searchPlaceholder))
    .replace(/\{\{SEARCH_ARIA\}\}/g, escapeHtml(i18n.searchAria))
    .replace(/\{\{BTN_EXPAND_ALL\}\}/g, escapeHtml(i18n.btnExpandAll))
    .replace(/\{\{BTN_COLLAPSE_ALL\}\}/g, escapeHtml(i18n.btnCollapseAll))
    .replace(/\{\{SECTIONS_HTML\}\}/g, sectionsHtml)
    .replace(/\/\*\s*\{\{VIEWER_JS\}\}\s*\*\//g, viewerJs)
    .replace(/\{\{VIEWER_JS\}\}/g, viewerJs);

  fs.writeFileSync(htmlFile, html, 'utf-8');
  const stats = fs.statSync(htmlFile);
  console.log(`[Build Success] Generated: ${htmlFile} (${(stats.size / 1024).toFixed(1)} KB)`);
}

/**
 * 全言語一括ビルド
 */
async function buildAll() {
  console.log('[Build] Starting unified/mdast Single-File Reference HTML generation...');

  if (!fs.existsSync(REFERENCES_DIR)) {
    console.error(`[Build Error] References directory not found: ${REFERENCES_DIR}`);
    process.exit(1);
  }

  const entries = fs.readdirSync(REFERENCES_DIR, { withFileTypes: true });
  const langDirs = entries.filter(e => e.isDirectory() && !e.name.startsWith('.')).map(e => e.name);

  if (langDirs.length === 0) {
    console.warn('[Build Warning] No language subdirectories found under docs/references/');
    return;
  }

  for (const lang of langDirs) {
    await buildLanguageEdition(lang);
  }

  console.log('[Build Complete] Successfully built all reference editions.');
}

// 実行
buildAll().catch(err => {
  console.error('[Build Fatal Error]', err);
  process.exit(1);
});
