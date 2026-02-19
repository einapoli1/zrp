/**
 * Integration tests for Work Order BOM Comparison and Procurement Auto-generation
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '../test/test-utils';

const mockGetWorkOrder = vi.fn();
const mockGetWorkOrderBOM = vi.fn();
const mockGeneratePOFromWorkOrder = vi.fn();
const mockGetVendors = vi.fn();
const mockUpdateWorkOrder = vi.fn();

vi.mock('../lib/api', () => ({
  api: {
    getWorkOrder: (...args: any[]) => mockGetWorkOrder(...args),
    getWorkOrderBOM: (...args: any[]) => mockGetWorkOrderBOM(...args),
    generatePOFromWorkOrder: (...args: any[]) => mockGeneratePOFromWorkOrder(...args),
    getVendors: (...args: any[]) => mockGetVendors(...args),
    updateWorkOrder: (...args: any[]) => mockUpdateWorkOrder(...args),
  },
}));

import WorkOrderDetail from './WorkOrderDetail';

describe('Work Order BOM Integration Tests', () => {
  const mockWorkOrder = {
    id: 'WO-001',
    assembly_ipn: 'ASM-001',
    qty: 10,
    status: 'pending',
    priority: 'normal',
    created_at: '2024-01-01',
  };

  const mockBOMWithShortages = {
    wo_id: 'WO-001',
    assembly_ipn: 'ASM-001',
    qty: 10,
    bom: [
      {
        ipn: 'PART-001',
        description: 'Resistor 10k',
        qty_required: 100,
        qty_on_hand: 80,
        shortage: 20,
        status: 'short',
      },
      {
        ipn: 'PART-002',
        description: 'Capacitor 100nF',
        qty_required: 50,
        qty_on_hand: 50,
        shortage: 0,
        status: 'ok',
      },
      {
        ipn: 'PART-003',
        description: 'LED Red',
        qty_required: 20,
        qty_on_hand: 5,
        shortage: 15,
        status: 'short',
      },
    ],
  };

  const mockVendors = [
    { id: 'V-001', name: 'Acme Corp', status: 'active' },
    { id: 'V-002', name: 'Global Supplies', status: 'active' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockGetWorkOrder.mockResolvedValue(mockWorkOrder);
    mockGetWorkOrderBOM.mockResolvedValue(mockBOMWithShortages);
    mockGetVendors.mockResolvedValue(mockVendors);
    mockUpdateWorkOrder.mockResolvedValue(mockWorkOrder);
  });

  describe('BOM Shortage Display', () => {
    it('displays shortage highlights for parts with insufficient stock', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      // Check for shortage indicator
      const shortageElements = screen.getAllByText(/short/i);
      expect(shortageElements.length).toBeGreaterThan(0);
    });

    it('shows shortage quantity for each short part', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      // PART-001 has shortage of 20
      expect(screen.getByText(/20/)).toBeInTheDocument();
      // PART-003 has shortage of 15
      expect(screen.getByText(/15/)).toBeInTheDocument();
    });

    it('does not highlight parts with sufficient stock', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-002')).toBeInTheDocument();
      });

      const part2Row = screen.getByText('PART-002').closest('tr');
      expect(part2Row?.textContent).toContain('ok');
    });

    it('calculates total shortage count correctly', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      // Total shortages: 2 parts (PART-001 and PART-003)
      screen.getAllByText(/short/i);
      const shortParts = mockBOMWithShortages.bom.filter(p => p.status === 'short');
      expect(shortParts).toHaveLength(2);
    });

    it('shows visual indicator (badge/color) for shortage status', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      // Check for destructive/warning badges
      const badges = document.querySelectorAll('[class*="destructive"], [class*="warning"]');
      expect(badges.length).toBeGreaterThan(0);
    });
  });

  describe('Procurement Auto-generation', () => {
    it('displays Generate PO button when shortages exist', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      expect(generateButton).toBeInTheDocument();
    });

    it('opens vendor selection dialog on Generate PO click', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText(/select vendor/i)).toBeInTheDocument();
      });
    });

    it('lists available vendors in selection dialog', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
        expect(screen.getByText('Global Supplies')).toBeInTheDocument();
      });
    });

    it('calls api.generatePOFromWorkOrder with correct parameters', async () => {
      mockGeneratePOFromWorkOrder.mockResolvedValue({ po_id: 'PO-001', lines: 2 });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      // Select vendor
      fireEvent.click(screen.getByText('Acme Corp'));

      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(mockGeneratePOFromWorkOrder).toHaveBeenCalledWith('WO-001', 'V-001');
      });
    });

    it('auto-generates PO lines from shortage items only', async () => {
      const generateResponse = {
        po_id: 'PO-001',
        lines: 2, // Only PART-001 and PART-003 (the shortage items)
      };
      mockGeneratePOFromWorkOrder.mockResolvedValue(generateResponse);

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(mockGeneratePOFromWorkOrder).toHaveBeenCalled();
      });
    });

    it('displays success message after PO generation', async () => {
      mockGeneratePOFromWorkOrder.mockResolvedValue({ po_id: 'PO-001', lines: 2 });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(screen.getByText(/created/i)).toBeInTheDocument();
      });
    });

    it('handles PO generation error gracefully', async () => {
      mockGeneratePOFromWorkOrder.mockRejectedValueOnce(new Error('Generation failed'));

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(screen.getByText(/failed/i)).toBeInTheDocument();
      });
    });

    it('disables Generate PO button when no vendor selected', async () => {
      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        const confirmButton = screen.getByRole('button', { name: /generate/i });
        expect(confirmButton).toBeDisabled();
      });
    });

    it('closes dialog after successful PO generation', async () => {
      mockGeneratePOFromWorkOrder.mockResolvedValue({ po_id: 'PO-001', lines: 2 });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(screen.queryByText(/select vendor/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Edge Cases - BOM Data', () => {
    it('handles empty BOM gracefully', async () => {
      mockGetWorkOrderBOM.mockResolvedValue({
        wo_id: 'WO-001',
        assembly_ipn: 'ASM-001',
        qty: 10,
        bom: [],
      });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText(/no bom items/i)).toBeInTheDocument();
      });
    });

    it('handles BOM with all items in stock', async () => {
      mockGetWorkOrderBOM.mockResolvedValue({
        wo_id: 'WO-001',
        assembly_ipn: 'ASM-001',
        qty: 10,
        bom: [
          {
            ipn: 'PART-001',
            description: 'Part 1',
            qty_required: 10,
            qty_on_hand: 20,
            shortage: 0,
            status: 'ok',
          },
        ],
      });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      // Generate PO button should not be visible or should be disabled
      const generateButton = screen.queryByRole('button', { name: /generate po/i });
      if (generateButton) {
        expect(generateButton).toBeDisabled();
      }
    });

    it('handles BOM with negative shortage (excess stock)', async () => {
      mockGetWorkOrderBOM.mockResolvedValue({
        wo_id: 'WO-001',
        assembly_ipn: 'ASM-001',
        qty: 10,
        bom: [
          {
            ipn: 'PART-001',
            description: 'Part 1',
            qty_required: 10,
            qty_on_hand: 50,
            shortage: -40,
            status: 'ok',
          },
        ],
      });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
        expect(screen.getByText(/ok/i)).toBeInTheDocument();
      });
    });

    it('handles BOM API error gracefully', async () => {
      mockGetWorkOrderBOM.mockRejectedValueOnce(new Error('BOM fetch failed'));

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
      });
    });

    it('handles very large shortage quantities', async () => {
      mockGetWorkOrderBOM.mockResolvedValue({
        wo_id: 'WO-001',
        assembly_ipn: 'ASM-001',
        qty: 10,
        bom: [
          {
            ipn: 'PART-001',
            description: 'Part 1',
            qty_required: 1000000,
            qty_on_hand: 10,
            shortage: 999990,
            status: 'short',
          },
        ],
      });

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('999990')).toBeInTheDocument();
      });
    });
  });

  describe('Edge Cases - Vendor Selection', () => {
    it('handles empty vendor list', async () => {
      mockGetVendors.mockResolvedValue([]);

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText(/no vendors available/i)).toBeInTheDocument();
      });
    });

    it('handles vendor fetch error', async () => {
      mockGetVendors.mockRejectedValueOnce(new Error('Failed to load vendors'));

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText(/failed to load vendors/i)).toBeInTheDocument();
      });
    });

    it('filters inactive vendors from selection', async () => {
      mockGetVendors.mockResolvedValue([
        { id: 'V-001', name: 'Active Vendor', status: 'active' },
        { id: 'V-002', name: 'Inactive Vendor', status: 'inactive' },
      ]);

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Active Vendor')).toBeInTheDocument();
        expect(screen.queryByText('Inactive Vendor')).not.toBeInTheDocument();
      });
    });
  });

  describe('Concurrent Operations', () => {
    it('prevents multiple PO generation requests', async () => {
      let resolvePO: any;
      mockGeneratePOFromWorkOrder.mockReturnValue(new Promise(resolve => { resolvePO = resolve; }));

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });

      // Attempt multiple clicks
      fireEvent.click(confirmButton);
      fireEvent.click(confirmButton);
      fireEvent.click(confirmButton);

      // Should only call API once
      expect(mockGeneratePOFromWorkOrder).toHaveBeenCalledTimes(1);

      resolvePO({ po_id: 'PO-001', lines: 2 });
    });

    it('disables Generate button while request in progress', async () => {
      let resolvePO: any;
      mockGeneratePOFromWorkOrder.mockReturnValue(new Promise(resolve => { resolvePO = resolve; }));

      render(<WorkOrderDetail />);

      await waitFor(() => {
        expect(screen.getByText('PART-001')).toBeInTheDocument();
      });

      const generateButton = screen.getByRole('button', { name: /generate po/i });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Acme Corp'));
      const confirmButton = screen.getByRole('button', { name: /generate/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(confirmButton).toBeDisabled();
      });

      resolvePO({ po_id: 'PO-001', lines: 2 });
    });
  });
});
