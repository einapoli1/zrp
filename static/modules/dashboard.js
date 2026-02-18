window.module_dashboard = {
  render: async (container) => {
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
      <div class="card">
        <h3 class="text-lg font-semibold mb-3">Welcome to ZRP</h3>
        <p class="text-gray-600">Zonit Resource Planning — your complete ERP for hardware electronics manufacturing. Use the sidebar to navigate between modules.</p>
      </div>
    `;
  }
};
