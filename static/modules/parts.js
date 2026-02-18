window.module_parts = {
  render: async (container) => {
    let currentCat = '';
    let searchQ = '';
    let page = 1;

    async function load() {
      const params = new URLSearchParams({ page, limit: 50 });
      if (currentCat) params.set('category', currentCat);
      if (searchQ) params.set('q', searchQ);
      const res = await api('GET', 'parts?' + params);
      const parts = res.data || [];
      const meta = res.meta || {};
      
      // Load categories
      const catRes = await api('GET', 'categories');
      const cats = catRes.data || [];

      container.innerHTML = `
        <div class="card mb-4">
          <div class="flex items-center justify-between mb-4">
            <div class="flex gap-2 flex-wrap">
              <button class="btn ${!currentCat ? 'btn-primary' : 'btn-secondary'}" onclick="window._partsSetCat('')">All</button>
              ${cats.map(c => `<button class="btn ${currentCat === c.id ? 'btn-primary' : 'btn-secondary'}" onclick="window._partsSetCat('${c.id}')">${c.name} (${c.count})</button>`).join('')}
            </div>
            <div class="flex gap-2">
              <input type="text" class="input w-48" placeholder="Search parts..." value="${searchQ}" onkeyup="if(event.key==='Enter'){window._partsSearch(this.value)}">
              <button class="btn btn-secondary" onclick="window._partsExport()">📥 CSV</button>
            </div>
          </div>
          ${parts.length === 0 ? '<p class="text-gray-500 text-center py-8">No parts found. Configure --pmDir to load gitplm CSVs.</p>' : `
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead><tr class="border-b text-left text-gray-500">
                <th class="pb-2 pr-4">IPN</th>
                <th class="pb-2 pr-4">Category</th>
                ${parts[0] && parts[0].fields ? Object.keys(parts[0].fields).filter(k=>k!=='_category'&&k!=='ipn'&&k!=='IPN'&&k!=='pn'&&k!=='PN').slice(0,5).map(k=>`<th class="pb-2 pr-4">${k}</th>`).join('') : ''}
              </tr></thead>
              <tbody>
                ${parts.map(p => `<tr class="table-row border-b border-gray-100" onclick="window._partsDetail('${p.ipn}')">
                  <td class="py-2 pr-4 font-mono text-blue-600">${p.ipn}</td>
                  <td class="py-2 pr-4">${p.fields?._category || ''}</td>
                  ${p.fields ? Object.entries(p.fields).filter(([k])=>k!=='_category'&&k!=='ipn'&&k!=='IPN'&&k!=='pn'&&k!=='PN').slice(0,5).map(([k,v])=>`<td class="py-2 pr-4 truncate max-w-[200px]">${v}</td>`).join('') : ''}
                </tr>`).join('')}
              </tbody>
            </table>
          </div>
          <div class="flex justify-between items-center mt-4 text-sm text-gray-500">
            <span>Showing ${parts.length} of ${meta.total || parts.length}</span>
            <div class="flex gap-2">
              ${page > 1 ? `<button class="btn btn-secondary" onclick="window._partsPage(${page-1})">← Prev</button>` : ''}
              ${(meta.total || 0) > page * 50 ? `<button class="btn btn-secondary" onclick="window._partsPage(${page+1})">Next →</button>` : ''}
            </div>
          </div>`}
        </div>
      `;
    }

    window._partsSetCat = (c) => { currentCat = c; page = 1; load(); };
    window._partsSearch = (q) => { searchQ = q; page = 1; load(); };
    window._partsPage = (p) => { page = p; load(); };
    window._partsDetail = async (ipn) => {
      const res = await api('GET', 'parts/' + ipn);
      const p = res.data;
      const fields = Object.entries(p.fields || {}).map(([k,v]) => `<div class="mb-2"><span class="label">${k}</span><div class="text-sm">${v}</div></div>`).join('');
      showModal('Part: ' + ipn, `<div class="font-mono text-lg text-blue-600 mb-4">${ipn}</div>${fields}`);
    };
    window._partsExport = () => { toast('CSV export: use gitplm CLI for full export'); };
    load();
  }
};
