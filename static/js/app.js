document.addEventListener('alpine:init', () => {
  Alpine.data('toast', () => ({
    visible: true,
    init() {
      setTimeout(() => { this.visible = false; }, 5000);
    }
  }));
});

document.addEventListener('keydown', (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.isContentEditable) return;

  const mod = e.metaKey || e.ctrlKey;
  if (mod && e.key === 'k') {
    e.preventDefault();
    const search = document.getElementById('global-search');
    if (search) search.focus();
  }
});

document.body.addEventListener('htmx:afterSwap', () => {
  if (typeof Alpine !== 'undefined') {
    document.querySelectorAll('[x-data]:not([x-init])').forEach(el => {
      Alpine.initTree(el);
    });
  }
});

// Sync session banner state every 60s to catch external changes (agent timers, etc.)
setInterval(() => {
  const banner = document.getElementById('session-banner');
  if (banner) {
    htmx.ajax('GET', '/partials/session-banner', { target: '#session-banner', swap: 'innerHTML' });
  }
}, 60000);
