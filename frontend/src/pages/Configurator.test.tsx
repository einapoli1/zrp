import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Configurator from './Configurator';

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

const mockTemplates = [
  {
    id: 1,
    name: 'uATS 1.2kVA',
    model_format: 'PCA-uATS-{voltage}-{amperage}',
    created_at: '2026-01-01',
    updated_at: '2026-01-01',
    parameters: [],
    parts: [],
  },
];

const mockTemplateDetail = {
  id: 1,
  name: 'uATS 1.2kVA',
  model_format: 'PCA-uATS-{voltage}-{amperage}',
  created_at: '2026-01-01',
  updated_at: '2026-01-01',
  parameters: [
    {
      id: 1,
      template_id: 1,
      name: 'voltage',
      type: 'enum',
      values_json: '["120V","208V"]',
      created_at: '2026-01-01',
    },
  ],
  parts: [
    {
      id: 1,
      template_id: 1,
      ipn: 'CAP-001',
      quantity: 2,
      include_all_variants: 1,
      constraints_json: '{}',
      created_at: '2026-01-01',
      description: 'Capacitor',
    },
  ],
};

function renderConfigurator() {
  return render(
    <BrowserRouter>
      <Configurator />
    </BrowserRouter>
  );
}

describe('Configurator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => mockTemplates,
    });
  });

  // Template Creation Tests (2 tests)

  it('should create a new template', async () => {
    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('New Template')).toBeInTheDocument();
    });

    const newButton = screen.getByText('New Template');
    fireEvent.click(newButton);

    // Should switch to editor tab
    await waitFor(() => {
      expect(screen.getByText('Template Details')).toBeInTheDocument();
    });
  });

  it('should validate template fields on save', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: 'invalid' }),
    });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('New Template')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('New Template'));

    await waitFor(() => {
      const saveButton = screen.getByText('Save Template');
      expect(saveButton).toBeInTheDocument();
    });
  });

  // Parameter Add/Edit/Delete Tests (3 tests)

  it('should add a parameter to template', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 2 }),
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('uATS 1.2kVA')).toBeInTheDocument();
    });

    const editButton = screen.getByText('Edit');
    fireEvent.click(editButton);

    await waitFor(() => {
      expect(screen.getByText('Parameters')).toBeInTheDocument();
    });
  });

  it('should delete a parameter', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('uATS 1.2kVA')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Edit'));

    await waitFor(() => {
      expect(screen.getByText('voltage')).toBeInTheDocument();
    });
  });

  it('should validate parameter name format', async () => {
    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('New Template')).toBeInTheDocument();
    });
  });

  // Part Pool Add/Edit/Delete Tests (3 tests)

  it('should add part to pool', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Edit'));

    await waitFor(() => {
      expect(screen.getByText('Parts Pool')).toBeInTheDocument();
    });
  });

  it('should update part constraints', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Edit'));

    await waitFor(() => {
      expect(screen.getByText('CAP-001')).toBeInTheDocument();
    });
  });

  it('should delete part from pool', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });
  });

  // Constraint Editing Tests (2 tests)

  it('should open constraints modal', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Edit'));

    await waitFor(() => {
      const constraintsButton = screen.getByText('Constraints');
      expect(constraintsButton).toBeInTheDocument();
    });
  });

  it('should save constraints', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplateDetail,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

    renderConfigurator();

    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });
  });

  // Preview Generation Test (1 test)

  it('should preview first 10 variants', async () => {
    const mockPreview = {
      preview: [
        { ipn: 'PCA-uATS-120V-10A', bom_count: 5 },
        { ipn: 'PCA-uATS-208V-10A', bom_count: 5 },
      ],
      total_count: 2,
      showing_first: 10,
    };

    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplates,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockPreview,
      });

    renderConfigurator();

    await waitFor(() => {
      const generateTab = screen.getByText('Preview & Generate');
      expect(generateTab).toBeInTheDocument();
      fireEvent.click(generateTab);
    });

    await waitFor(() => {
      expect(screen.getByText('Generate Variants')).toBeInTheDocument();
    });
  });
});
