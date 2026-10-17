document.addEventListener('DOMContentLoaded', () => {
  // 1. トースト要素の準備
  const copiedToastText = document.body.getAttribute('data-copied-toast') || 'Copied to clipboard';
  let toast = document.getElementById('copyToast');
  if (!toast) {
    toast = document.createElement('div');
    toast.id = 'copyToast';
    toast.className = 'copy-toast';
    toast.setAttribute('role', 'status');
    toast.setAttribute('aria-live', 'polite');
    toast.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="#98c379" aria-hidden="true"><path d="M9 16.2L4.8 12l-1.4 1.4L9 19 21 7l-1.4-1.4L9 16.2z"/></svg><span>${copiedToastText}</span>`;
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
      showCopyToast(copiedToastText + ': ' + (text.length > 25 ? text.slice(0, 25) + '...' : text));
    } catch (err) {
      console.error('Copy failed:', err);
    }
  }

  // インラインコードのクリックコピー
  document.querySelectorAll('code.clickable-code').forEach(el => {
    el.addEventListener('click', (e) => {
      e.stopPropagation();
      const code = el.getAttribute('data-copy') || el.textContent;
      copyText(code);
    });
  });

  // コードブロックヘッダーのコピーボタン
  document.querySelectorAll('.btn-copy-code').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const wrapper = btn.closest('.code-block-wrapper');
      if (wrapper) {
        const codeEl = wrapper.querySelector('pre code');
        if (codeEl) {
          copyText(codeEl.textContent);
        }
      }
    });
  });

  // 3. インクリメンタル高速検索
  const searchInput = document.getElementById('searchInput');
  const searchStatusBar = document.getElementById('searchStatusBar');
  const searchResultText = document.getElementById('searchResultText');
  const accordions = document.querySelectorAll('.dads-accordion');

  if (searchInput) {
    searchInput.addEventListener('input', () => {
      const query = searchInput.value.trim().toLowerCase();
      if (!query) {
        accordions.forEach((acc, idx) => {
          acc.style.display = '';
          // 検索解除時は先頭4つのみ開く
          acc.open = idx < 4;
        });
        if (searchStatusBar) searchStatusBar.style.display = 'none';
        return;
      }

      let matchCount = 0;
      accordions.forEach(acc => {
        const keywords = (acc.getAttribute('data-keywords') || '').toLowerCase();
        const text = acc.textContent.toLowerCase();
        if (keywords.includes(query) || text.includes(query)) {
          acc.style.display = '';
          acc.open = true; // マッチしたアコーディオンは自動展開
          matchCount++;
        } else {
          acc.style.display = 'none';
        }
      });

      if (searchStatusBar && searchResultText) {
        searchStatusBar.style.display = 'block';
        searchResultText.textContent = `${matchCount} 件のセクションが一致しました`;
      }
    });
  }

  // 4. 一括展開・折りたたみ
  const btnExpandAll = document.getElementById('btnExpandAll');
  const btnCollapseAll = document.getElementById('btnCollapseAll');

  if (btnExpandAll) {
    btnExpandAll.addEventListener('click', () => {
      accordions.forEach(acc => {
        if (acc.style.display !== 'none') acc.open = true;
      });
    });
  }

  if (btnCollapseAll) {
    btnCollapseAll.addEventListener('click', () => {
      accordions.forEach(acc => {
        if (acc.style.display !== 'none') acc.open = false;
      });
    });
  }

  // 5. 目次スクロール連動 (IntersectionObserver)
  const tocLinks = document.querySelectorAll('.sidebar-toc .toc-link, .sidebar-toc .toc-sublink');
  const observedTargets = document.querySelectorAll('main.main-content h2, .dads-accordion');

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const id = entry.target.id;
        if (!id) return;
        tocLinks.forEach(link => {
          if (link.getAttribute('href') === '#' + id) {
            link.classList.add('active');
            // 親 details.toc-group があれば自動展開
            const parentDetails = link.closest('details.toc-group');
            if (parentDetails) parentDetails.open = true;
          } else {
            link.classList.remove('active');
          }
        });
      }
    });
  }, {
    rootMargin: '-80px 0px -70% 0px',
    threshold: 0
  });

  observedTargets.forEach(el => observer.observe(el));

  // 目次リンククリック時に該当アコーディオンを自動展開
  document.querySelectorAll('.toc-link, .toc-sublink').forEach(link => {
    link.addEventListener('click', () => {
      const targetId = (link.getAttribute('href') || '').replace('#', '');
      const targetEl = document.getElementById(targetId);
      if (targetEl && targetEl.classList.contains('dads-accordion')) {
        targetEl.open = true;
      }
    });
  });

  // 6. モバイルドロワー開閉
  const btnHamburger = document.getElementById('btnHamburger');
  const btnCloseDrawer = document.getElementById('btnCloseDrawer');
  const mobileDrawer = document.getElementById('mobileDrawer');
  const drawerOverlay = document.getElementById('drawerOverlay');

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

  document.querySelectorAll('.mobile-drawer .toc-link, .mobile-drawer .toc-sublink').forEach(link => {
    link.addEventListener('click', closeDrawer);
  });
});