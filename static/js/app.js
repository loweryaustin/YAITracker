document.addEventListener('alpine:init', () => {
  Alpine.data('timer', () => ({
    running: false,
    elapsed: 0,
    interval: null,

    start() {
      this.running = true;
      this.interval = setInterval(() => { this.elapsed++; }, 1000);
    },

    stop() {
      this.running = false;
      clearInterval(this.interval);
      this.interval = null;
    },

    reset() {
      this.stop();
      this.elapsed = 0;
    },

    get display() {
      const h = Math.floor(this.elapsed / 3600);
      const m = Math.floor((this.elapsed % 3600) / 60);
      const s = this.elapsed % 60;
      return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
    }
  }));

  Alpine.data('sidebar', () => ({
    collapsed: localStorage.getItem('sidebar-collapsed') === 'true',
    toggle() {
      this.collapsed = !this.collapsed;
      localStorage.setItem('sidebar-collapsed', this.collapsed);
    }
  }));

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
