window.module_dashboard = {
  _refreshTimer: null,
  render: async (container) => {
    // Clear any previous timer
    if (window.module_dashboard._refreshTimer) {
      clearInterval(window.module_dashboard._refreshTimer);
    }

    const data = await fetch('/api/v1/dashboard').then(r => r.json());
    const cards = [
      { label: 'Open ECOs', value: data.open_ecos, icon: '🔄', color: 'blue', route: 'ecos' },
      { label: 'Low Stock Items', value: data.low_stock, icon: '📊', color: 'red', route: 'inventory' },
      { label: 'Open POs', value: data.open_pos, icon: '🛒', color: 'purple', route: 'procurement' },
      { label: 'Active Work Orders', value: data.active_wos, icon: '⚙️', color: 'yellow', route: 'workorders' },
      { label: 'Open NCRs', value: data.open_ncrs, icon: '⚠️', color: 'orange', route: 'ncr' },
      { label: 'Open RMAs', value: data.open_rmas, icon: '🔧', color: 'teal', route: 'rma' },
      { label: 'Total Parts', value: data.total_parts, icon: '📦', color: 'gray', route: 'parts' },
      { label: 'Total Devices', value: data.total_devices, icon: '📱', color: 'green', route: 'devices' },
    ];

    container.innerHTML = `
      <div id="low-stock-banner"></div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        ${cards.map(c => `
          <div class="card cursor-pointer hover:shadow-md transition-shadow" onclick="navigate('${c.route}')">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-gray-500">${c.label}</p>
                <p class="text-3xl font-bold text-gray-900 mt-1">${c.value}</p>
              </div>
              <span class="text-3xl">${c.icon}</span>
            </div>
          </div>
        `).join('')}
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <div class="card"><h3 class="text-sm font-semibold text-gray-500 mb-3">ECOs by Status</h3><canvas id="chart-ecos"></canvas></div>
        <div class="card"><h3 class="text-sm font-semibold text-gray-500 mb-3">Work Orders by Status</h3><canvas id="chart-wos"></canvas></div>
        <div class="card"><h3 class="text-sm font-semibold text-gray-500 mb-3">Inventory Value (Top 10)</h3><canvas id="chart-inv"></canvas></div>
      </div>

      <div class="card">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold">Recent Activity</h3>
          <span class="text-xs text-gray-400">Auto-refreshes every 30s</span>
        </div>
        <div id="activity-feed" class="space-y-3"></div>
      </div>
    `;

    // Load low stock alerts
    loadLowStockAlerts();

    // Load charts
    loadCharts();

    // Load activity feed
    loadActivityFeed();

    // Auto-refresh activity feed
    window.module_dashboard._refreshTimer = setInterval(loadActivityFeed, 30000);
  }
};

const moduleIcons = {
  eco: { icon: '🔄', color: 'bg-blue-100 text-blue-700' },
  inventory: { icon: '📊', color: 'bg-green-100 text-green-700' },
  workorder: { icon: '⚙️', color: 'bg-yellow-100 text-yellow-700' },
  ncr: { icon: '⚠️', color: 'bg-orange-100 text-orange-700' },
  device: { icon: '📱', color: 'bg-emerald-100 text-emerald-700' },
  document: { icon: '📄', color: 'bg-indigo-100 text-indigo-700' },
  vendor: { icon: '🏭', color: 'bg-purple-100 text-purple-700' },
  po: { icon: '🛒', color: 'bg-pink-100 text-pink-700' },
  rma: { icon: '🔧', color: 'bg-teal-100 text-teal-700' },
  quote: { icon: '💰', color: 'bg-amber-100 text-amber-700' },
  test: { icon: '🧪', color: 'bg-cyan-100 text-cyan-700' },
  firmware: { icon: '💾', color: 'bg-violet-100 text-violet-700' },
};

function relativeTime(dateStr) {
  const now = new Date();
  const d = new Date(dateStr + (dateStr.includes('Z') ? '' : 'Z'));
  const diff = Math.floor((now - d) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return Math.floor(diff / 60) + ' min ago';
  if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
  if (diff < 604800) return Math.floor(diff / 86400) + 'd ago';
  return d.toLocaleDateString();
}

async function loadActivityFeed() {
  try {
    const res = await fetch('/api/v1/audit?limit=20').then(r => r.json());
    const entries = res.data || [];
    const feed = document.getElementById('activity-feed');
    if (!feed) return;

    if (entries.length === 0) {
      feed.innerHTML = '<p class="text-gray-400 text-sm">No activity yet.</p>';
      return;
    }

    feed.innerHTML = entries.map(e => {
      const mod = moduleIcons[e.module] || { icon: '📋', color: 'bg-gray-100 text-gray-700' };
      return `
        <div class="flex items-start gap-3">
          <div class="flex-shrink-0 w-8 h-8 rounded-full ${mod.color} flex items-center justify-center text-sm">${mod.icon}</div>
          <div class="flex-1 min-w-0">
            <p class="text-sm text-gray-800"><span class="font-medium">${e.username}</span> <span class="text-gray-500">${e.action}</span> ${e.summary || e.record_id}</p>
            <p class="text-xs text-gray-400">${relativeTime(e.created_at)}</p>
          </div>
        </div>
      `;
    }).join('');
  } catch(err) {
    console.error('Failed to load activity feed:', err);
  }
}

async function loadCharts() {
  try {
    const res = await fetch('/api/v1/dashboard/charts').then(r => r.json());
    const d = res.data || {};

    // ECOs by Status - horizontal bar
    const ecoCtx = document.getElementById('chart-ecos');
    if (ecoCtx) {
      const ecos = d.ecos_by_status || {};
      new Chart(ecoCtx, {
        type: 'bar',
        data: {
          labels: ['Draft', 'Review', 'Approved', 'Implemented'],
          datasets: [{
            data: [ecos.draft||0, ecos.review||0, ecos.approved||0, ecos.implemented||0],
            backgroundColor: ['#9ca3af', '#fbbf24', '#34d399', '#60a5fa'],
            borderRadius: 4,
          }]
        },
        options: {
          indexAxis: 'y',
          plugins: { legend: { display: false } },
          scales: { x: { beginAtZero: true, ticks: { stepSize: 1 } } }
        }
      });
    }

    // Work Orders - donut
    const woCtx = document.getElementById('chart-wos');
    if (woCtx) {
      const wos = d.wos_by_status || {};
      new Chart(woCtx, {
        type: 'doughnut',
        data: {
          labels: ['Open', 'In Progress', 'Completed'],
          datasets: [{
            data: [wos.open||0, wos.in_progress||0, wos.completed||0],
            backgroundColor: ['#3b82f6', '#f59e0b', '#10b981'],
          }]
        },
        options: {
          plugins: { legend: { position: 'bottom', labels: { boxWidth: 12 } } }
        }
      });
    }

    // Inventory Value - bar
    const invCtx = document.getElementById('chart-inv');
    if (invCtx) {
      const inv = d.inventory_value || [];
      new Chart(invCtx, {
        type: 'bar',
        data: {
          labels: inv.map(i => i.ipn),
          datasets: [{
            data: inv.map(i => i.value),
            backgroundColor: '#6366f1',
            borderRadius: 4,
          }]
        },
        options: {
          indexAxis: 'y',
          plugins: { legend: { display: false } },
          scales: { x: { beginAtZero: true } }
        }
      });
    }
  } catch(err) {
    console.error('Failed to load charts:', err);
  }
}

async function loadLowStockAlerts() {
  try {
    const res = await fetch('/api/v1/dashboard/lowstock').then(r => r.json());
    const items = res.data || [];
    const banner = document.getElementById('low-stock-banner');
    if (!banner) return;

    if (items.length === 0) {
      banner.innerHTML = '';
      return;
    }

    banner.innerHTML = `
      <div class="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
        <div class="flex items-start gap-3">
          <span class="text-red-600 text-xl">🚨</span>
          <div>
            <h4 class="text-red-800 font-semibold text-sm">Low Stock Alert</h4>
            <p class="text-red-700 text-sm mt-1">
              ${items.map(i => `<span class="font-mono">${i.ipn}</span> (${i.qty_on_hand} on hand, reorder at ${i.reorder_point})`).join(' · ')}
            </p>
          </div>
        </div>
      </div>
    `;
  } catch(err) {
    console.error('Failed to load low stock alerts:', err);
  }
}
