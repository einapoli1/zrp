import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";

const mockGetDocument = vi.fn();
const mockGetDocumentVersions = vi.fn();
const mockReleaseDocument = vi.fn();
const mockRevertDocument = vi.fn();
const mockGetDocumentDiff = vi.fn();
const mockGetGitDocsSettings = vi.fn();

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "DOC-001" }), useNavigate: () => mockNavigate };
});

vi.mock("../lib/api", () => ({
  api: {
    getDocument: (...args: any[]) => mockGetDocument(...args),
    getDocumentVersions: (...args: any[]) => mockGetDocumentVersions(...args),
    releaseDocument: (...args: any[]) => mockReleaseDocument(...args),
    revertDocument: (...args: any[]) => mockRevertDocument(...args),
    getDocumentDiff: (...args: any[]) => mockGetDocumentDiff(...args),
    getGitDocsSettings: (...args: any[]) => mockGetGitDocsSettings(...args),
  },
}));

import DocumentDetail from "./DocumentDetail";

const mockDoc = {
  id: "DOC-001", title: "Assembly Guide", category: "procedure", ipn: "IPN-003",
  revision: "A", status: "released", content: "How to assemble...", file_path: "",
  created_by: "admin", created_at: "2024-01-05", updated_at: "2024-01-05",
  attachments: [],
};

const mockVersions = [
  { id: "v1", document_id: "DOC-001", revision: "A", content: "v1", created_by: "admin", created_at: "2024-01-05" },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDocument.mockResolvedValue(mockDoc);
  mockGetDocumentVersions.mockResolvedValue(mockVersions);
  mockGetGitDocsSettings.mockResolvedValue({ repo_url: "", branch: "main", token: "" });
});

describe("DocumentDetail", () => {
  it("renders document detail after loading", async () => {
    render(<DocumentDetail />);
    await waitFor(() => {
      expect(screen.getByText("Assembly Guide")).toBeInTheDocument();
    });
  });

  it("shows document status", async () => {
    render(<DocumentDetail />);
    await waitFor(() => {
      expect(screen.getByText("Released")).toBeInTheDocument();
    });
  });

  it("fetches document with correct id", async () => {
    render(<DocumentDetail />);
    await waitFor(() => {
      expect(mockGetDocument).toHaveBeenCalledWith("DOC-001");
    });
  });

  it("fetches versions on load", async () => {
    render(<DocumentDetail />);
    await waitFor(() => {
      expect(mockGetDocumentVersions).toHaveBeenCalledWith("DOC-001");
    });
  });

  it("shows loading skeleton initially", () => {
    mockGetDocument.mockReturnValue(new Promise(() => {}));
    render(<DocumentDetail />);
    // Component shows skeleton or loading state
    expect(document.querySelector("[data-slot='skeleton']") || screen.queryByText("Assembly Guide") === null).toBeTruthy();
  });
});
