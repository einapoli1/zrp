window.module_inventory = {
  render: async (container) => {
    let showLow = false;
    async function load() {
      const res = await api('GET', 'inventory' + (showLow ? '?low_stock=true' : ''));
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Inventory</h2>
          <div class="flex gap-2">
            <button class="btn ${showLow?'btn-danger':'btn-secondary'}" onclick="window._invToggleLow()">
              ${showLow?'🔴 Low Stock Only':'Show Low Stock'}
            </button>
            <button class="btn btn-primary" onclick="window._invReceive()">+ Quick Receive</button>
          </div>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">IPN</th><th class="pb-2">On Hand</th><th class="pb-2">Reserved</th><th class="pb-2">Available</th><th class="pb-2">Location</th><th class="pb-2">Reorder Point</th>
        </tr></thead><tbody>
          ${items.map(i => {
            const low = i.reorder_point > 0 && i.qty_on_hand <= i.reorder_point;
            return `<tr class="table-row border-b border-gray-100 ${low?'bg-red-50':''}" onclick="window._invHistory('${i.ipn}')">
              <td class="py-2 font-mono text-blue-600">${i.ipn}</td>
              <td class="py-2 ${low?'text-red-600 font-bold':''}">${i.qty_on_hand}</td>
              <td class="py-2">${i.qty_reserved}</td>
              <td class="py-2">${i.qty_on_hand - i.qty_reserved}</td>
              <td class="py-2">${i.location||''}</td>
              <td class="py-2">${i.reorder_point}</td>
            </tr>`;
          }).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No inventory records</p>':''}
      </div>`;
    }
    window._invToggleLow = () => { showLow = !showLow; load(); };
    window._invReceive = () => showModal('Quick Receive', `<div class="space-y-3">
      <div><label class="label">IPN</label><input class="input" data-field="ipn"></div>
      <div><label class="label">Quantity</label><input class="input" data-field="qty" type="number"></div>
      <div><label class="label">Reference (PO#, etc)</label><input class="input" data-field="reference"></div>
      <div><label class="label">Notes</label><input class="input" data-field="notes"></div>
    </div>`, async (o) => {
      const v = getModalValues(o);
      try { await api('POST', 'inventory/transact', {ipn:v.ipn, type:'receive', qty:parseFloat(v.qty), reference:v.reference, notes:v.notes}); toast('Stock received'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._invHistory = async (ipn) => {
      const res = await api('GET', 'inventory/' + ipn + '/history');
      const txns = res.data || [];
      showModal('Transaction History: ' + ipn, `<table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
        <th class="pb-2 text-left">Type</th><th class="pb-2 text-left">Qty</th><th class="pb-2 text-left">Reference</th><th class="pb-2 text-left">Date</th>
      </tr></thead><tbody>
        ${txns.map(t => `<tr class="border-b border-gray-100"><td class="py-1">${badge(t.type)}</td><td class="py-1">${t.qty}</td><td class="py-1">${t.reference||''}</td><td class="py-1 text-gray-500">${t.created_at?.substring(0,16)}</td></tr>`).join('')}
      </tbody></table>${txns.length===0?'<p class="text-gray-400 text-center py-4">No transactions</p>':''}`);
    };
    load();
  }
};
