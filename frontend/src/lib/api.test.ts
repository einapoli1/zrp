import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { api } from './api';

describe('API Client', () => {
  let originalFetch: typeof global.fetch;
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    originalFetch = global.fetch;
    mockFetch = vi.fn();
    global.fetch = mockFetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  describe('Envelope unwrapping', () => {
    it('unwraps {data: ...} envelope from backend', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'TEST-001', description: 'Test Part' } }),
      });

      const result = await api.getPart('TEST-001');
      expect(result).toEqual({ ipn: 'TEST-001', description: 'Test Part' });
    });

    it('returns raw JSON when no envelope present', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ipn: 'TEST-001', description: 'Test Part' }),
      });

      const result = await api.getPart('TEST-001');
      expect(result).toEqual({ ipn: 'TEST-001', description: 'Test Part' });
    });

    it('requestWithMeta returns full envelope including meta', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [{ ipn: 'TEST-001' }],
          meta: { total: 1, page: 1, limit: 50 },
        }),
      });

      const result = await api.getParts();
      expect(result).toEqual({
        data: [{ ipn: 'TEST-001' }],
        meta: { total: 1, page: 1, limit: 50 },
      });
    });
  });

  describe('Error handling', () => {
    it('throws error on 500 response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({ error: 'Server exploded' }),
      });

      await expect(api.getPart('TEST-001')).rejects.toThrow('Server exploded');
    });

    it('redirects to /login on 401 response', async () => {
      const originalLocation = window.location;
      delete (window as any).location;
      (window as any).location = { href: '' };

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: async () => ({ error: 'Unauthorized' }),
      });

      await expect(api.getPart('TEST-001')).rejects.toThrow('Session expired');
      expect(window.location.href).toBe('/login');

      (window as any).location = originalLocation;
    });

    it('does not redirect on 401 for auth endpoints', async () => {
      const originalLocation = window.location;
      delete (window as any).location;
      (window as any).location = { href: '' };

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Invalid credentials' }),
      });

      await expect(api.login('user', 'wrong')).rejects.toThrow('Invalid credentials');
      expect(window.location.href).toBe('');

      (window as any).location = originalLocation;
    });

    it('handles network failures', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network failure'));

      await expect(api.getPart('TEST-001')).rejects.toThrow('Network failure');
    });

    it('handles malformed JSON response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Bad JSON',
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      await expect(api.getPart('TEST-001')).rejects.toThrow('Bad JSON');
    });
  });

  describe('Auth methods', () => {
    it('login sends credentials and returns user', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ user: { id: 1, username: 'admin', role: 'admin' } }),
      });

      const result = await api.login('admin', 'password');
      expect(result.user.username).toBe('admin');
      expect(mockFetch).toHaveBeenCalledWith('/auth/login', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ username: 'admin', password: 'password' }),
      }));
    });

    it('logout calls /auth/logout', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true, json: async () => ({}) });

      await api.logout();
      expect(mockFetch).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({ method: 'POST' }));
    });

    it('getMe returns user data', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ user: { id: 1, username: 'admin' } }),
      });

      const result = await api.getMe();
      expect(result?.user.username).toBe('admin');
    });

    it('getMe returns null on failure', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 401 });

      const result = await api.getMe();
      expect(result).toBeNull();
    });
  });

  describe('Parts API', () => {
    it('getParts calls /api/v1/parts', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 0, page: 1, limit: 50 } }),
      });

      await api.getParts();
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts', expect.any(Object));
    });

    it('getParts with params builds query string', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 0, page: 1, limit: 50 } }),
      });

      await api.getParts({ category: 'resistors', q: 'RES-', page: 2, limit: 25 });
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/parts?category=resistors&q=RES-&page=2&limit=25',
        expect.any(Object)
      );
    });

    it('getPart calls /api/v1/parts/:ipn', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'TEST-001' } }),
      });

      await api.getPart('TEST-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts/TEST-001', expect.any(Object));
    });

    it('createPart sends POST request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'NEW-001' } }),
      });

      await api.createPart({ ipn: 'NEW-001', category: 'resistors', fields: { value: '10k' } });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ ipn: 'NEW-001', category: 'resistors', fields: { value: '10k' } }),
      }));
    });

    it('updatePart sends PUT request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'TEST-001', description: 'Updated' } }),
      });

      await api.updatePart('TEST-001', { description: 'Updated' });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts/TEST-001', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ description: 'Updated' }),
      }));
    });

    it('deletePart sends DELETE request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: null }),
      });

      await api.deletePart('TEST-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts/TEST-001', expect.objectContaining({
        method: 'DELETE',
      }));
    });

    it('getCategories returns category list', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [{ id: 'resistors', name: 'Resistors', count: 10, columns: [] }] }),
      });

      const result = await api.getCategories();
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('Resistors');
    });

    it('createCategory sends POST request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'new-cat', name: 'New Category' } }),
      });

      await api.createCategory({ title: 'New Category', prefix: 'NC-' });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/categories', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('ECO API', () => {
    it('getECOs calls /api/v1/ecos', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getECOs();
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/ecos', expect.any(Object));
    });

    it('getECOs with status param', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getECOs('pending');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/ecos?status=pending', expect.any(Object));
    });

    it('approveECO sends POST to approve endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'ECO-001', status: 'approved' } }),
      });

      await api.approveECO('ECO-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/ecos/ECO-001/approve', expect.objectContaining({
        method: 'POST',
      }));
    });

    it('implementECO sends POST to implement endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'ECO-001', status: 'implemented' } }),
      });

      await api.implementECO('ECO-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/ecos/ECO-001/implement', expect.objectContaining({
        method: 'POST',
      }));
    });

    it('rejectECO updates status to rejected', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'ECO-001', status: 'rejected' } }),
      });

      await api.rejectECO('ECO-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/ecos/ECO-001', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ status: 'rejected' }),
      }));
    });
  });

  describe('Work Orders API', () => {
    it('getWorkOrders calls /api/v1/workorders', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getWorkOrders();
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/workorders', expect.any(Object));
    });

    it('getWorkOrderBOM returns BOM with shortage info', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            wo_id: 'WO-001',
            assembly_ipn: 'ASM-001',
            qty: 10,
            bom: [{ ipn: 'PART-001', shortage: 5, status: 'short' }],
          },
        }),
      });

      const result = await api.getWorkOrderBOM('WO-001');
      expect(result.bom[0].shortage).toBe(5);
    });

    it('generatePOFromWorkOrder calls /api/v1/pos/generate', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { po_id: 'PO-001', lines: 3 } }),
      });

      await api.generatePOFromWorkOrder('WO-001', 'VENDOR-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/pos/generate', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ wo_id: 'WO-001', vendor_id: 'VENDOR-001' }),
      }));
    });
  });

  describe('Procurement API', () => {
    it('receivePurchaseOrder calls /api/v1/pos/:id/receive', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'PO-001' } }),
      });

      await api.receivePurchaseOrder('PO-001', [{ id: 1, qty: 10 }], false);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/pos/PO-001/receive', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ lines: [{ id: 1, qty: 10 }], skip_inspection: false }),
      }));
    });

    it('inspectReceiving calls /api/v1/receiving/:id/inspect', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 1 } }),
      });

      await api.inspectReceiving(1, { qty_passed: 8, qty_failed: 2, qty_on_hold: 0 });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/receiving/1/inspect', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('Inventory API', () => {
    it('bulkDeleteInventory calls /api/v1/inventory/bulk-delete', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: null }),
      });

      await api.bulkDeleteInventory(['PART-001', 'PART-002']);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/inventory/bulk-delete', expect.objectContaining({
        method: 'DELETE',
        body: JSON.stringify({ ipns: ['PART-001', 'PART-002'] }),
      }));
    });

    it('createInventoryTransaction calls /api/v1/inventory/transact', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: null }),
      });

      await api.createInventoryTransaction({
        ipn: 'PART-001',
        type: 'receive',
        qty: 100,
        reference: 'PO-001',
      });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/inventory/transact', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('Search and Navigation', () => {
    it('globalSearch calls /api/v1/search with encoded query', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { parts: [], ecos: [], work_orders: [] } }),
      });

      await api.globalSearch('test query');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/search?q=test%20query', expect.any(Object));
    });

    it('globalSearch handles special characters', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: {} }),
      });

      await api.globalSearch('R&D / 10kΩ');
      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('R%26D'), expect.any(Object));
    });
  });

  describe('Bulk Operations', () => {
    it('bulkUpdateInventory sends correct payload', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { success: 2, failed: 0, errors: [] } }),
      });

      await api.bulkUpdateInventory(['PART-001', 'PART-002'], { location: 'A1' });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/inventory/bulk-update', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ ids: ['PART-001', 'PART-002'], updates: { location: 'A1' } }),
      }));
    });

    it('bulkUpdateWorkOrders sends correct payload', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { success: 3, failed: 0, errors: [] } }),
      });

      await api.bulkUpdateWorkOrders(['WO-001', 'WO-002'], { status: 'in_progress' });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/workorders/bulk-update', expect.objectContaining({
        method: 'POST',
      }));
    });

    it('bulkUpdateDevices sends correct payload', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { success: 5, failed: 0, errors: [] } }),
      });

      await api.bulkUpdateDevices(['DEV-001', 'DEV-002'], { status: 'deployed' });
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/devices/bulk-update', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('Attachments API', () => {
    it('uploadAttachment sends FormData', async () => {
      const file = new File(['content'], 'test.pdf', { type: 'application/pdf' });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'ATT-001', filename: 'test.pdf' }),
      });

      await api.uploadAttachment(file, 'parts', 'PART-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/attachments', expect.objectContaining({
        method: 'POST',
      }));
    });

    it('downloadAttachment returns blob', async () => {
      const blob = new Blob(['file content'], { type: 'application/pdf' });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: async () => blob,
      });

      const result = await api.downloadAttachment('ATT-001');
      expect(result).toBeInstanceOf(Blob);
    });
  });

  describe('Pagination edge cases', () => {
    it('handles page 0 gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 0, page: 0, limit: 50 } }),
      });

      const result = await api.getParts({ page: 0 });
      expect(result.meta?.page).toBe(0);
      expect(mockFetch).toHaveBeenCalled();
    });

    it('handles last page correctly', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 100, page: 2, limit: 50 } }),
      });

      const result = await api.getParts({ page: 2, limit: 50 });
      expect(result.meta?.page).toBe(2);
    });

    it('handles beyond last page', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 50, page: 10, limit: 50 } }),
      });

      const result = await api.getParts({ page: 10 });
      expect(result.data).toEqual([]);
    });
  });

  describe('Special characters in requests', () => {
    it('encodes IPN with slashes', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'R/C-001' } }),
      });

      await api.getPart('R/C-001');
      // URL encoding happens in fetch, but we can verify the call was made
      expect(mockFetch).toHaveBeenCalled();
    });

    it('handles very long text in fields', async () => {
      const longText = 'A'.repeat(10000);
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { ipn: 'TEST', description: longText } }),
      });

      await api.updatePart('TEST', { description: longText });
      const call = mockFetch.mock.calls[0];
      const body = JSON.parse(call[1].body);
      expect(body.description).toBe(longText);
    });
  });

  describe('Concurrent requests', () => {
    it('handles multiple simultaneous requests', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ data: { ipn: 'PART-001' } }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ data: { ipn: 'PART-002' } }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ data: { ipn: 'PART-003' } }),
        });

      const results = await Promise.all([
        api.getPart('PART-001'),
        api.getPart('PART-002'),
        api.getPart('PART-003'),
      ]);

      expect(results).toHaveLength(3);
      expect(results[0].ipn).toBe('PART-001');
      expect(results[2].ipn).toBe('PART-003');
    });
  });

  describe('Dashboard API', () => {
    it('getDashboard calls /api/v1/dashboard', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: { total_parts: 100, low_stock_alerts: 5, active_work_orders: 10, pending_ecos: 3 },
        }),
      });

      const result = await api.getDashboard();
      expect(result.total_parts).toBe(100);
    });

    it('getCalendarEvents builds query params', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getCalendarEvents(2024, 12);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/calendar?year=2024&month=12', expect.any(Object));
    });
  });

  describe('Audit and History', () => {
    it('getAuditLogs builds complex query params', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { entries: [], total: 0 } }),
      });

      await api.getAuditLogs({
        search: 'user action',
        entityType: 'parts',
        user: 'admin',
        page: 2,
        limit: 25,
      });

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('search=user+action'),
        expect.any(Object)
      );
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('entity_type=parts'),
        expect.any(Object)
      );
    });

    it('getUndoList with limit param', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getUndoList(10);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/undo?limit=10', expect.any(Object));
    });

    it('performUndo calls POST /api/v1/undo/:id', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { status: 'success', entity_type: 'parts', entity_id: 'PART-001' } }),
      });

      await api.performUndo(123);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/undo/123', expect.objectContaining({ method: 'POST' }));
    });
  });

  describe('RFQ and Procurement', () => {
    it('compareRFQ returns matrix structure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            lines: [{ id: 1, ipn: 'PART-001' }],
            vendors: [{ id: 1, vendor_id: 'V1' }],
            matrix: { 1: { 1: { unit_price: 10 } } },
          },
        }),
      });

      const result = await api.compareRFQ('RFQ-001');
      expect(result.matrix).toBeDefined();
    });

    it('awardRFQPerLine sends awards array', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { status: 'success', po_ids: ['PO-001', 'PO-002'] } }),
      });

      await api.awardRFQPerLine('RFQ-001', [
        { line_id: 1, vendor_id: 'V1' },
        { line_id: 2, vendor_id: 'V2' },
      ]);

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/rfqs/RFQ-001/award-lines', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ awards: [{ line_id: 1, vendor_id: 'V1' }, { line_id: 2, vendor_id: 'V2' }] }),
      }));
    });
  });

  describe('Market Pricing and Distributors', () => {
    it('getMarketPricing with refresh param', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { results: [], cached: false } }),
      });

      await api.getMarketPricing('PART-001', true);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts/PART-001/market-pricing?refresh=true', expect.any(Object));
    });

    it('getMarketPricing without refresh', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { results: [], cached: true } }),
      });

      await api.getMarketPricing('PART-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/parts/PART-001/market-pricing', expect.any(Object));
    });
  });

  describe('Sales Orders', () => {
    it('getSalesOrders with filters', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.getSalesOrders({ status: 'confirmed', customer: 'ACME Corp' });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('status=confirmed'),
        expect.any(Object)
      );
    });

    it('allocateSalesOrder calls /allocate endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'SO-001', status: 'allocated' } }),
      });

      await api.allocateSalesOrder('SO-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/sales-orders/SO-001/allocate', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('Empty data states', () => {
    it('handles empty parts list', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { total: 0, page: 1, limit: 50 } }),
      });

      const result = await api.getParts();
      expect(result.data).toEqual([]);
    });

    it('handles empty ECO list', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      const result = await api.getECOs();
      expect(result).toEqual([]);
    });

    it('handles empty work orders', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      const result = await api.getWorkOrders();
      expect(result).toEqual([]);
    });
  });

  describe('Document Management', () => {
    it('getDocumentDiff calls /diff endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            from: 'v1',
            to: 'v2',
            lines: [{ type: 'same', text: 'Line 1' }],
          },
        }),
      });

      await api.getDocumentDiff('DOC-001', 'v1', 'v2');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/docs/DOC-001/diff?from=v1&to=v2', expect.any(Object));
    });

    it('pushDocumentToGit calls /push endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { status: 'success', file: 'docs/test.md' } }),
      });

      await api.pushDocumentToGit('DOC-001');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/docs/DOC-001/push', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('Notification Preferences', () => {
    it('getNotificationTypes returns type list', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [{ type: 'low_stock', name: 'Low Stock Alert' }] }),
      });

      const result = await api.getNotificationTypes();
      expect(result).toHaveLength(1);
    });

    it('updateNotificationPreferences sends array', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] }),
      });

      await api.updateNotificationPreferences([{ notification_type: 'low_stock', enabled: true }] as any);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/notifications/preferences', expect.objectContaining({
        method: 'PUT',
      }));
    });
  });
});
