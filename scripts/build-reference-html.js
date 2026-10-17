#!/usr/bin/env node

/**
 * build-reference-html.js
 * 
 * docs/references/{lang}/reference.{lang}.md を原典（SSOT）として読み込み、
 * 完全自己完結型（Single-File）HTML（reference.{lang}.html）を自動生成するビルドスクリプト。
 * 
 * 外部依存ゼロ（Node.js 標準ライブラリ fs, path のみ使用）
 * DADS（デジタル庁デザインシステム）デザイントークン・アクセシビリティ（WCAG 2.1 AA+）準拠
 */

const fs = require('fs');
const path = require('path');

const ROOT_DIR = path.resolve(__dirname, '..');
const REFERENCES_DIR = path.join(ROOT_DIR, 'docs', 'references');

// 共通インライン CSS (DADS 準拠・完全自己完結)
// ※アプローチ2: 表コンテナの常時シャドウを完全撤去し、クリーンなボーダー仕様を採用
const INLINE_CSS = `
  :root {
    --color-primary: #0055ff;
    --color-primary-hover: #0040cc;
    --color-primary-active: #0033aa;
    --color-primary-text: #0044cc;
    --color-primary-bg: #f2f6fc;
    --color-text-main: #1a1a1c;
    --color-text-muted: #535359;
    --color-bg-base: #f9f9fb;
    --color-bg-surface: #ffffff;
    --color-bg-subtle: #f2f2f5;
    --color-border: #d0d0d8;
    --color-border-subtle: #e4e4eb;
    --color-border-dark: #1a1a1c;
    --color-focus: #0055ff;
    --color-code-bg: #21252b;
    --color-code-text: #abb2bf;
    --color-notice-bg: #e6f0ff;
    --color-notice-border: #0055ff;
    --font-family-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans JP", "Hiragino Sans", "Meiryo", sans-serif;
    --font-family-mono: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace;
    --radius-sm: 4px;
    --radius-md: 8px;
    --space-1: 4px;
    --space-2: 8px;
    --space-3: 12px;
    --space-4: 16px;
    --space-5: 24px;
    --space-6: 32px;
    --space-8: 48px;
    --sidebar-width: 280px;
    --header-height: 56px;
  }

  *, *::before, *::after {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  html {
    scroll-behavior: smooth;
    font-size: 16px;
  }

  body {
    font-family: var(--font-family-sans);
    color: var(--color-text-main);
    background-color: var(--color-bg-base);
    line-height: 1.7;
    -webkit-font-smoothing: antialiased;
  }

  :focus-visible {
    outline: 4px solid var(--color-focus);
    outline-offset: 2px;
  }

  .skip-link {
    position: absolute;
    top: -9999px;
    left: var(--space-4);
    background: var(--color-primary);
    color: #fff;
    padding: var(--space-2) var(--space-4);
    z-index: 9999;
    font-weight: 700;
    text-decoration: none;
    border-radius: var(--radius-sm);
  }
  .skip-link:focus {
    top: var(--space-4);
  }

  .site-header {
    position: sticky;
    top: 0;
    z-index: 100;
    height: var(--header-height);
    background-color: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--space-5);
  }

  .header-brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .logo-title {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--color-text-main);
    text-decoration: none;
  }

  .version-badge {
    font-size: 0.75rem;
    font-weight: 600;
    background-color: var(--color-primary-bg);
    color: var(--color-primary-text);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    border: 1px solid rgba(0, 85, 255, 0.2);
  }

  .btn-hamburger {
    display: none;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    background: transparent;
    border: none;
    cursor: pointer;
    color: var(--color-text-main);
    border-radius: var(--radius-sm);
  }
  .btn-hamburger:hover {
    background-color: var(--color-bg-subtle);
  }

  .page-container {
    display: flex;
    max-width: 1440px;
    margin: 0 auto;
    position: relative;
  }

  .sidebar-toc {
    width: var(--sidebar-width);
    flex-shrink: 0;
    height: calc(100vh - var(--header-height));
    position: sticky;
    top: var(--header-height);
    overflow-y: auto;
    padding: var(--space-5) var(--space-4);
    border-right: 1px solid var(--color-border);
    background-color: var(--color-bg-surface);
  }

  .toc-title {
    font-size: 0.875rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted);
    margin-bottom: var(--space-3);
  }

  .toc-list {
    list-style: none;
  }
  .toc-item {
    margin-bottom: var(--space-1);
  }
  .toc-link {
    display: block;
    padding: var(--space-1) var(--space-2);
    font-size: 0.875rem;
    color: var(--color-text-muted);
    text-decoration: none;
    border-radius: var(--radius-sm);
    line-height: 1.4;
    transition: background-color 0.15s, color 0.15s;
  }
  .toc-link:hover {
    color: var(--color-text-main);
    background-color: var(--color-bg-subtle);
  }
  .toc-link.active {
    color: var(--color-primary-text);
    background-color: var(--color-primary-bg);
    font-weight: 700;
    border-left: 3px solid var(--color-primary);
    padding-left: calc(var(--space-2) - 3px);
  }

  .main-content {
    flex: 1;
    min-width: 0;
    padding: var(--space-6) var(--space-6) var(--space-8);
  }

  .search-wrapper {
    margin-bottom: var(--space-5);
  }

  .search-bar-inner {
    position: relative;
    width: 100%;
  }

  .search-input {
    width: 100%;
    height: 48px;
    padding: 0 var(--space-5) 0 44px;
    font-size: 1rem;
    font-family: inherit;
    border: 2px solid var(--color-border);
    border-radius: var(--radius-md);
    background-color: var(--color-bg-surface);
    color: var(--color-text-main);
    transition: border-color 0.2s, box-shadow 0.2s;
  }
  .search-input:focus {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgba(0, 85, 255, 0.2);
    outline: none;
  }

  .search-icon {
    position: absolute;
    left: 14px;
    top: 50%;
    transform: translateY(-50%);
    width: 20px;
    height: 20px;
    color: var(--color-text-muted);
    pointer-events: none;
  }

  .search-status-bar {
    display: none;
    margin-top: var(--space-2);
    font-size: 0.875rem;
    color: var(--color-text-muted);
  }

  .search-summary-text {
    font-weight: 600;
  }

  .search-count-badge {
    background-color: var(--color-primary-bg);
    color: var(--color-primary-text);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
    font-weight: 700;
  }

  .section-toolbar {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-bottom: var(--space-4);
  }

  .btn-action-sm {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    min-height: 36px;
    padding: 0 var(--space-3);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--color-text-main);
    background-color: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: background-color 0.15s, border-color 0.15s;
  }
  .btn-action-sm:hover {
    background-color: var(--color-bg-subtle);
    border-color: #b5b5be;
  }

  details.dads-accordion {
    background-color: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    margin-bottom: var(--space-5);
    overflow: hidden;
  }

  summary.accordion-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4);
    background-color: var(--color-bg-subtle);
    cursor: pointer;
    list-style: none;
    -webkit-user-select: none;
    user-select: none;
    border-left: 5px solid var(--color-primary);
  }
  summary.accordion-summary::-webkit-details-marker {
    display: none;
  }
  summary.accordion-summary:hover {
    background-color: #ebebee;
  }

  .accordion-title {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--color-text-main);
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .accordion-icon {
    width: 20px;
    height: 20px;
    fill: var(--color-text-muted);
    transition: transform 0.2s ease;
  }
  details.dads-accordion[open] .accordion-icon {
    transform: rotate(180deg);
    fill: var(--color-primary);
  }

  .accordion-content {
    padding: var(--space-5) var(--space-5) var(--space-6);
    border-top: 1px solid var(--color-border);
  }

  h1.dads-h1 {
    font-size: 1.875rem;
    font-weight: 700;
    line-height: 1.3;
    margin-bottom: var(--space-4);
    padding-bottom: var(--space-3);
    border-bottom: 3px solid var(--color-primary);
  }

  h3.dads-h3 {
    font-size: 1.125rem;
    font-weight: 700;
    line-height: 1.5;
    margin-top: var(--space-5);
    margin-bottom: var(--space-3);
    padding-bottom: var(--space-1);
    border-bottom: 1px solid var(--color-border-subtle);
  }

  .lead-text {
    font-size: 1.0625rem;
    margin-bottom: var(--space-5);
    color: var(--color-text-muted);
  }

  /* テーブルコンテナ (アプローチ2: クリーン枠線のみ、不要なシャドウ完全撤去) */
  .table-container {
    width: 100%;
    overflow-x: auto;
    margin: var(--space-3) 0 var(--space-5);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background-color: var(--color-bg-surface);
  }

  table.dads-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
    font-size: 0.9375rem;
    line-height: 1.6;
  }

  /* テーブルヘッダー: 下部2px黒ボーダー ＋ 縦罫線 */
  table.dads-table thead th {
    background-color: var(--color-bg-subtle);
    color: var(--color-text-main);
    font-weight: 700;
    padding: var(--space-3) var(--space-4);
    border-bottom: 2px solid var(--color-border-dark);
    border-right: 1px solid var(--color-border);
    white-space: nowrap;
    vertical-align: top;
  }
  table.dads-table thead th:last-child {
    border-right: none;
  }

  /* テーブルセル: 下部細罫線 ＋ 縦罫線 */
  table.dads-table tbody td {
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--color-border-subtle);
    border-right: 1px solid var(--color-border-subtle);
    vertical-align: top;
    color: var(--color-text-main);
  }
  table.dads-table tbody td:last-child {
    border-right: none;
  }
  table.dads-table tbody tr:last-child td {
    border-bottom: none;
  }
  table.dads-table tbody tr:hover td {
    background-color: var(--color-primary-bg);
  }

  .col-rank {
    width: 60px;
    text-align: center;
    font-weight: 600;
    color: var(--color-text-muted);
  }

  .col-operator {
    white-space: nowrap;
    font-family: var(--font-family-mono);
    font-weight: 700;
    color: var(--color-primary-text);
  }

  /* インラインコード & クリックコピー対応 */
  code.clickable-code {
    font-family: var(--font-family-mono);
    font-size: 0.875rem;
    background-color: var(--color-bg-subtle);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-primary-text);
    cursor: pointer;
    display: inline-block;
    transition: background-color 0.15s, border-color 0.15s, transform 0.1s;
    user-select: all;
    -webkit-user-select: all;
  }
  code.clickable-code:hover {
    background-color: #e0ecff;
    border-color: #99c2ff;
    transform: translateY(-1px);
  }
  code.clickable-code:active {
    transform: translateY(0);
  }

  /* コードブロック (REPLカードスタイル) */
  .code-block-wrapper {
    position: relative;
    margin: var(--space-4) 0;
    border-radius: var(--radius-md);
    background-color: var(--color-code-bg);
    border: 1px solid #3e4451;
    overflow: hidden;
  }
  .code-block-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-2) var(--space-4);
    background-color: #1e2227;
    border-bottom: 1px solid #2c313a;
  }
  .code-block-title {
    font-size: 0.75rem;
    font-family: var(--font-family-mono);
    font-weight: 700;
    color: #9da5b4;
    text-transform: uppercase;
  }
  pre.dads-code-pre {
    padding: var(--space-4);
    overflow-x: auto;
    font-family: var(--font-family-mono);
    font-size: 0.875rem;
    line-height: 1.6;
    color: var(--color-code-text);
  }

  /* コピートースト通知 */
  .copy-toast {
    position: fixed;
    bottom: var(--space-6);
    right: var(--space-6);
    background-color: #1a1a1c;
    color: #ffffff;
    padding: var(--space-3) var(--space-5);
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    opacity: 0;
    transform: translateY(10px);
    transition: opacity 0.2s, transform 0.2s;
    pointer-events: none;
    z-index: 1000;
  }
  .copy-toast.show {
    opacity: 1;
    transform: translateY(0);
  }

  /* モバイル対応ドロワー */
  .drawer-overlay {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.4);
    z-index: 200;
    backdrop-filter: blur(2px);
  }
  .mobile-drawer {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 300px;
    max-width: 80%;
    background-color: var(--color-bg-surface);
    z-index: 201;
    padding: var(--space-5) var(--space-4);
    overflow-y: auto;
    transform: translateX(-100%);
    transition: transform 0.25s ease;
    box-shadow: 2px 0 16px rgba(0, 0, 0, 0.15);
  }
  .mobile-drawer.open {
    transform: translateX(0);
  }

  @media (max-width: 768px) {
    .btn-hamburger {
      display: inline-flex;
    }
    .sidebar-toc {
      display: none;
    }
    .main-content {
      padding: var(--space-4) var(--space-4) var(--space-6);
    }
    .drawer-overlay, .mobile-drawer {
      display: block;
    }
  }
`;

// 共通インライン JavaScript (Vanilla JS・完全自己完結)
const INLINE_JS = `
  document.addEventListener('DOMContentLoaded', () => {
    // 1. トースト要素の準備
    let toast = document.getElementById('copyToast');
    if (!toast) {
      toast = document.createElement('div');
      toast.id = 'copyToast';
      toast.className = 'copy-toast';
      toast.setAttribute('role', 'status');
      toast.setAttribute('aria-live', 'polite');
      toast.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="#98c379" aria-hidden="true"><path d="M9 16.2L4.8 12l-1.4 1.4L9 19 21 7l-1.4-1.4L9 16.2z"/></svg><span>クリップボードにコピーしました</span>';
      document.body.appendChild(toast);
    }

    let toastTimer = null;
    function showCopyToast(msg) {
      if (msg) toast.querySelector('span').textContent = msg;
      toast.classList.add('show');
      clearTimeout(toastTimer);
      toastTimer = setTimeout(() => {
        toast.classList.remove('show');
      }, 1800);
    }

    // 2. コピー処理 (Clipboard API + フォールバック)
    async function copyText(text) {
      if (!text) return;
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text);
        } else {
          const textarea = document.createElement('textarea');
          textarea.value = text;
          textarea.style.position = 'fixed';
          textarea.style.opacity = '0';
          document.body.appendChild(textarea);
          textarea.select();
          document.execCommand('copy');
          document.body.removeChild(textarea);
        }
        showCopyToast('コピーしました: ' + (text.length > 25 ? text.slice(0, 25) + '...' : text));
      } catch (err) {
        console.error('Copy failed:', err);
      }
    }

    // 3. クリックコピーイベントの登録 (code.clickable-code)
    document.querySelectorAll('code.clickable-code').forEach(el => {
      el.addEventListener('click', () => {
        const text = el.getAttribute('data-copy') || el.textContent.trim();
        copyText(text);
      });
    });

    // 4. アコーディオン一括開閉
    const btnExpandAll = document.getElementById('btnExpandAll');
    const btnCollapseAll = document.getElementById('btnCollapseAll');
    const accordions = document.querySelectorAll('details.dads-accordion');

    if (btnExpandAll) {
      btnExpandAll.addEventListener('click', () => {
        accordions.forEach(acc => acc.open = true);
      });
    }
    if (btnCollapseAll) {
      btnCollapseAll.addEventListener('click', () => {
        accordions.forEach(acc => acc.open = false);
      });
    }

    // 5. インクリメンタル検索
    const searchInput = document.getElementById('searchInput');
    const searchStatusBar = document.getElementById('searchStatusBar');
    const searchResultText = document.getElementById('searchResultText');

    if (searchInput) {
      searchInput.addEventListener('input', (e) => {
        const query = e.target.value.trim().toLowerCase();
        if (!query) {
          accordions.forEach(acc => acc.style.display = '');
          if (searchStatusBar) searchStatusBar.style.display = 'none';
          return;
        }

        let hitCount = 0;
        accordions.forEach(acc => {
          const keywords = (acc.getAttribute('data-keywords') || '').toLowerCase();
          const text = acc.textContent.toLowerCase();
          if (keywords.includes(query) || text.includes(query)) {
            acc.style.display = '';
            acc.open = true;
            hitCount++;
          } else {
            acc.style.display = 'none';
          }
        });

        if (searchStatusBar && searchResultText) {
          searchStatusBar.style.display = 'block';
          searchResultText.innerHTML = 'キーワード: <span class="search-query-text">"' + query + '"</span> （一致セクション: <span class="search-count-badge">' + hitCount + ' 件</span>）';
        }
      });
    }

    // 6. スクロールスパイ (現在地TOCハイライト)
    const tocLinks = document.querySelectorAll('.toc-link');
    const observedSections = document.querySelectorAll('details[id]');

    if ('IntersectionObserver' in window && observedSections.length > 0) {
      const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            const id = entry.target.getAttribute('id');
            tocLinks.forEach(link => {
              if (link.getAttribute('href') === '#' + id) {
                link.classList.add('active');
              } else {
                link.classList.remove('active');
              }
            });
          }
        });
      }, {
        root: null,
        rootMargin: '-80px 0px -70% 0px',
        threshold: 0
      });

      observedSections.forEach(sec => observer.observe(sec));
    }

    // 7. モバイルドロワー開閉
    const btnHamburger = document.getElementById('btnHamburger');
    const drawerOverlay = document.getElementById('drawerOverlay');
    const mobileDrawer = document.getElementById('mobileDrawer');
    const btnCloseDrawer = document.getElementById('btnCloseDrawer');

    function openDrawer() {
      if (mobileDrawer) mobileDrawer.classList.add('open');
      if (drawerOverlay) drawerOverlay.style.display = 'block';
    }
    function closeDrawer() {
      if (mobileDrawer) mobileDrawer.classList.remove('open');
      if (drawerOverlay) drawerOverlay.style.display = 'none';
    }

    if (btnHamburger) btnHamburger.addEventListener('click', openDrawer);
    if (drawerOverlay) drawerOverlay.addEventListener('click', closeDrawer);
    if (btnCloseDrawer) btnCloseDrawer.addEventListener('click', closeDrawer);
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') closeDrawer();
    });

    document.querySelectorAll('.mobile-drawer .toc-link').forEach(link => {
      link.addEventListener('click', closeDrawer);
    });
  });
`;

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
 * インラインテキスト整形 (インラインコード、太字、HTMLエスケープ)
 */
function formatInlineText(text) {
  if (!text) return '';
  // 1. `code` を <code class="clickable-code"> に変換
  let formatted = text.replace(/`([^`]+)`/g, (match, p1) => {
    const cleanText = p1.trim();
    return `<code class="clickable-code" data-copy="${escapeHtml(cleanText)}" title="クリックしてコピー">${escapeHtml(cleanText)}</code>`;
  });
  // 2. **太字** を <strong> に変換
  formatted = formatted.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  return formatted;
}

/**
 * テーブル行を分割（インラインコードや数式 $...$ 内のパイプ | を保護）
 */
function splitTableRow(line) {
  let inBacktick = false;
  let inDollar = false;
  let cleanLine = '';
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === '`') inBacktick = !inBacktick;
    else if (ch === '$') inDollar = !inDollar;
    
    if (ch === '|' && (inBacktick || inDollar)) {
      cleanLine += '\uE000'; // プライベートUnicode文字で一時退避
    } else {
      cleanLine += ch;
    }
  }
  return cleanLine.split('|').slice(1, -1).map(c => c.replace(/\uE000/g, '|').trim());
}

/**
 * Markdown パースと HTML 構築
 */
function parseMarkdownToReferenceHtml(markdownContent, lang) {
  const lines = markdownContent.split(/\r?\n/);
  let title = 'ihd Complete Reference';
  let leadText = '';
  
  const tocItems = [];
  const sections = []; // 各セクション: { id, title, contentLines, keywords }
  let currentSection = null;
  let inTable = false;
  let tableHeader = [];
  let tableRows = [];
  let inCodeBlock = false;
  let codeBlockLang = '';
  let codeBlockLines = [];

  function flushTable() {
    if (!inTable) return '';
    let html = '<div class="table-container"><table class="dads-table"><thead><tr>';
    tableHeader.forEach((th, idx) => {
      html += `<th scope="col">${th}</th>`;
    });
    html += '</tr></thead><tbody>';

    tableRows.forEach(row => {
      html += '<tr>';
      row.forEach((cell, idx) => {
        let cellClass = '';
        if (idx === 0 && (/^\d+$/.test(cell.trim()) || cell.trim() === '順位')) {
          cellClass = 'col-rank';
        } else if (idx === 1 || (idx === 0 && tableHeader[0] && /関数|形式|演算子/i.test(tableHeader[0]))) {
          cellClass = 'col-operator';
        }
        
        const formattedCell = formatInlineText(cell);
        html += `<td${cellClass ? ` class="${cellClass}"` : ''}>${formattedCell}</td>`;
      });
      html += '</tr>';
    });

    html += '</tbody></table></div>';
    inTable = false;
    tableHeader = [];
    tableRows = [];
    return html;
  }

  function flushCodeBlock() {
    if (!inCodeBlock) return '';
    const code = escapeHtml(codeBlockLines.join('\n'));
    let html = `<div class="code-block-wrapper"><div class="code-block-header"><span class="code-block-title">${codeBlockLang || 'CONSOLE'}</span></div><pre class="dads-code-pre"><code>${code}</code></pre></div>`;
    inCodeBlock = false;
    codeBlockLang = '';
    codeBlockLines = [];
    return html;
  }

  let lineIdx = 0;
  while (lineIdx < lines.length) {
    const line = lines[lineIdx];

    // 1. コードブロック処理
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        if (currentSection) currentSection.contentHtml += flushCodeBlock();
      } else {
        if (inTable) {
          if (currentSection) currentSection.contentHtml += flushTable();
        }
        inCodeBlock = true;
        codeBlockLang = line.replace('```', '').trim();
      }
      lineIdx++;
      continue;
    }

    if (inCodeBlock) {
      codeBlockLines.push(line);
      lineIdx++;
      continue;
    }

    // 2. テーブル処理
    if (line.trim().startsWith('|') && line.trim().endsWith('|')) {
      const cells = splitTableRow(line);
      // セパレーター行 (|:---|---:| 等)
      if (cells.every(c => /^:?-+:?$/.test(c))) {
        lineIdx++;
        continue;
      }
      if (!inTable) {
        inTable = true;
        tableHeader = cells;
      } else {
        tableRows.push(cells);
      }
      lineIdx++;
      continue;
    } else if (inTable) {
      if (currentSection) currentSection.contentHtml += flushTable();
    }

    // 3. タイトル (# 見出し)
    if (line.startsWith('# ') && !line.startsWith('## ')) {
      title = line.replace('# ', '').trim();
      lineIdx++;
      continue;
    }

    // 4. 目次セクション (## 目次) はスキップ (自動生成するため)
    if (line.startsWith('## 目次') || line.startsWith('## Table of Contents')) {
      // 次のセクション開始まで読み飛ばす
      lineIdx++;
      while (lineIdx < lines.length && !lines[lineIdx].startsWith('## ')) {
        lineIdx++;
      }
      continue;
    }

    // 5. 大見出し (## ) -> 新しいアコーディオンセクション
    if (line.startsWith('## ')) {
      const secTitle = line.replace('## ', '').trim();
      const secId = 'sec-' + secTitle.toLowerCase().replace(/[^a-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]+/g, '-').replace(/^-|-$/g, '');

      currentSection = {
        id: secId,
        title: secTitle,
        contentHtml: '',
        keywords: [secTitle]
      };
      sections.push(currentSection);
      tocItems.push({ id: secId, title: secTitle });
      lineIdx++;
      continue;
    }

    // 6. 中見出し (### )
    if (line.startsWith('### ')) {
      const subTitle = line.replace('### ', '').trim();
      if (currentSection) {
        currentSection.keywords.push(subTitle);
        currentSection.contentHtml += `<h3 class="dads-h3">${escapeHtml(subTitle)}</h3>`;
      }
      lineIdx++;
      continue;
    }

    // 7. 本文テキスト / リスト
    if (line.trim() === '---') {
      lineIdx++;
      continue;
    }

    if (line.trim() !== '') {
      if (!currentSection && !leadText) {
        leadText = line.trim();
      } else if (currentSection) {
        if (line.startsWith('- ')) {
          currentSection.contentHtml += `<p class="bullet-item">• ${formatInlineText(escapeHtml(line.slice(2)))}</p>`;
        } else {
          currentSection.contentHtml += `<p class="body-p">${formatInlineText(escapeHtml(line))}</p>`;
        }
      }
    }

    lineIdx++;
  }

  if (inTable && currentSection) {
    currentSection.contentHtml += flushTable();
  }
  if (inCodeBlock && currentSection) {
    currentSection.contentHtml += flushCodeBlock();
  }

  // TOC リスト HTML
  let tocHtml = '<ul class="toc-list">';
  tocItems.forEach(item => {
    tocHtml += `<li class="toc-item"><a class="toc-link" href="#${item.id}">${escapeHtml(item.title)}</a></li>`;
  });
  tocHtml += '</ul>';

  // セクションアコーディオン HTML
  let sectionsHtml = '';
  sections.forEach((sec, idx) => {
    // 最初の数セクションは初期状態で open
    const isOpen = idx < 3 ? ' open' : '';
    sectionsHtml += `
      <details class="dads-accordion" id="${sec.id}"${isOpen} data-keywords="${escapeHtml(sec.keywords.join(' '))}">
        <summary class="accordion-summary" aria-label="${escapeHtml(sec.title)} を開閉">
          <span class="accordion-title">${escapeHtml(sec.title)}</span>
          <svg class="accordion-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg>
        </summary>
        <div class="accordion-content">
          ${sec.contentHtml}
        </div>
      </details>
    `;
  });

  return `<!DOCTYPE html>
<html lang="${lang}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="description" content="${escapeHtml(leadText || title)}">
  <title>${escapeHtml(title)}</title>
  <style>${INLINE_CSS}</style>
</head>
<body>
  <a class="skip-link" href="#mainContent">本文へスキップ</a>

  <header class="site-header">
    <div class="header-brand">
      <button class="btn-hamburger" id="btnHamburger" aria-label="目次メニューを開く">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z"/></svg>
      </button>
      <a class="logo-title" href="#">ihd Reference</a>
      <span class="version-badge">v2.18.1</span>
    </div>
  </header>

  <!-- モバイル用スライドイン・ドロワー -->
  <div class="drawer-overlay" id="drawerOverlay"></div>
  <aside class="mobile-drawer" id="mobileDrawer" aria-label="モバイル目次ナビゲーション">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
      <span style="font-weight: 700; font-size: 1.125rem;">目次</span>
      <button id="btnCloseDrawer" style="background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--color-text-main);">&times;</button>
    </div>
    ${tocHtml}
  </aside>

  <div class="page-container">
    <nav class="sidebar-toc" id="sidebarToc" aria-label="デスクトップ目次ナビゲーション">
      <h2 class="toc-title">目次</h2>
      ${tocHtml}
    </nav>

    <main class="main-content" id="mainContent">
      <h1 class="dads-h1">${escapeHtml(title)}</h1>
      ${leadText ? `<p class="lead-text">${escapeHtml(leadText)}</p>` : ''}

      <div class="search-wrapper">
        <div class="search-bar-inner">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 14z"/></svg>
          <input type="search" class="search-input" id="searchInput" placeholder="関数名、演算子、キーワードで高速検索..." aria-label="リファレンス内を検索">
        </div>
        <div class="search-status-bar" id="searchStatusBar">
          <span id="searchResultText"></span>
        </div>
      </div>

      <div class="section-toolbar">
        <button type="button" class="btn-action-sm" id="btnExpandAll">すべて展開</button>
        <button type="button" class="btn-action-sm" id="btnCollapseAll">すべて折りたたむ</button>
      </div>

      <div class="sections-container">
        ${sectionsHtml}
      </div>
    </main>
  </div>

  <script>${INLINE_JS}</script>
</body>
</html>`;
}

/**
 * メインビルド処理
 */
function buildAll() {
  console.log('[Build] Starting Single-File Reference HTML generation...');

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

  let successCount = 0;

  for (const lang of langDirs) {
    const langDir = path.join(REFERENCES_DIR, lang);
    const mdFile = path.join(langDir, `reference.${lang}.md`);
    const htmlFile = path.join(langDir, `reference.${lang}.html`);

    if (!fs.existsSync(mdFile)) {
      console.warn(`[Build Warning] Markdown file not found: ${mdFile}, skipping...`);
      continue;
    }

    console.log(`[Build] Processing ${lang} edition: ${mdFile}`);
    const mdContent = fs.readFileSync(mdFile, 'utf-8');
    const htmlContent = parseMarkdownToReferenceHtml(mdContent, lang);

    fs.writeFileSync(htmlFile, htmlContent, 'utf-8');
    const stats = fs.statSync(htmlFile);
    console.log(`[Build Success] Generated: ${htmlFile} (${(stats.size / 1024).toFixed(1)} KB)`);
    successCount++;
  }

  console.log(`[Build Complete] Successfully built ${successCount} HTML reference(s).`);
}

// 実行
buildAll();
