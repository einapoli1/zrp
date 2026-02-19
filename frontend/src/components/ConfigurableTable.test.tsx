import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "../test/test-utils";
import { ConfigurableTable, type ColumnDef } from "./ConfigurableTable";

interface TestRow {
  id: string;
  name: string;
  value: number;
  extra: string;
}

const testData: TestRow[] = [
  { id: "1", name: "Alpha", value: 10, extra: "x" },
  { id: "2", name: "Beta", value: 20, extra: "y" },
];

const sortTestData: TestRow[] = [
  { id: "1", name: "Charlie", value: 30, extra: "a" },
  { id: "2", name: "Alpha", value: 10, extra: "c" },
  { id: "3", name: "Beta", value: 20, extra: "b" },
];

const testColumns: ColumnDef<TestRow>[] = [
  { id: "name", label: "Name", accessor: (r) => r.name, sortValue: (r) => r.name },
  { id: "value", label: "Value", accessor: (r) => r.value, sortValue: (r) => r.value },
  { id: "extra", label: "Extra", accessor: (r) => r.extra, defaultVisible: false, sortValue: (r) => r.extra },
];

beforeEach(() => {
  localStorage.clear();
});

function getCellTexts(column: number): string[] {
  const rows = screen.getAllByRole("row").slice(1); // skip header
  return rows.map((row) => {
    const cells = row.querySelectorAll("td");
    return cells[column]?.textContent || "";
  });
}

describe("ConfigurableTable", () => {
  it("renders visible columns and data", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Value")).toBeInTheDocument();
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
    expect(screen.queryByText("Extra")).not.toBeInTheDocument();
  });

  it("hides columns with defaultVisible=false", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    expect(screen.queryByText("x")).not.toBeInTheDocument();
  });

  it("shows empty message when no data", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={[]}
        rowKey={(r) => r.id}
        emptyMessage="Nothing here"
      />
    );
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("calls onRowClick when row is clicked", () => {
    const onClick = vi.fn();
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
        onRowClick={onClick}
      />
    );
    fireEvent.click(screen.getByText("Alpha"));
    expect(onClick).toHaveBeenCalledWith(testData[0]);
  });

  it("persists column state to localStorage", () => {
    render(
      <ConfigurableTable
        tableName="persist-test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    const stored = localStorage.getItem("zrp-columns-persist-test");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed).toHaveLength(3);
    expect(parsed[0].id).toBe("name");
    expect(parsed[2].visible).toBe(false);
  });

  it("restores column state from localStorage", () => {
    const state = [
      { id: "name", visible: false },
      { id: "value", visible: true },
      { id: "extra", visible: true },
    ];
    localStorage.setItem("zrp-columns-restore-test", JSON.stringify(state));

    render(
      <ConfigurableTable
        tableName="restore-test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    expect(screen.getByText("x")).toBeInTheDocument();
    expect(screen.getByText("y")).toBeInTheDocument();
  });

  it("has settings gear button for column config", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    const settingsBtn = screen.getAllByRole("button").find(
      (btn) => btn.querySelector("svg.lucide-settings-2")
    );
    expect(settingsBtn).toBeTruthy();
  });

  it("renders leading column when provided", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
        leadingColumn={{
          header: <span>Check</span>,
          cell: (r) => <input type="checkbox" data-testid={`check-${r.id}`} />,
        }}
      />
    );
    expect(screen.getByText("Check")).toBeInTheDocument();
    expect(screen.getByTestId("check-1")).toBeInTheDocument();
    expect(screen.getByTestId("check-2")).toBeInTheDocument();
  });

  describe("sorting", () => {
    it("sorts ascending by string column on first click", () => {
      render(
        <ConfigurableTable
          tableName="sort-asc-str"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Alpha", "Beta", "Charlie"]);
    });

    it("sorts descending by string column on second click", () => {
      render(
        <ConfigurableTable
          tableName="sort-desc-str"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Charlie", "Beta", "Alpha"]);
    });

    it("sorts ascending by numeric column", () => {
      render(
        <ConfigurableTable
          tableName="sort-asc-num"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["10", "20", "30"]);
    });

    it("sorts descending by numeric column", () => {
      render(
        <ConfigurableTable
          tableName="sort-desc-num"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Value"));
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["30", "20", "10"]);
    });

    it("clears sort on third click (asc → desc → none cycle)", () => {
      render(
        <ConfigurableTable
          tableName="sort-cycle"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      // Original order: Charlie, Alpha, Beta
      expect(getCellTexts(0)).toEqual(["Charlie", "Alpha", "Beta"]);
    });

    it("sorts empty/null values to end regardless of direction", () => {
      interface NullableRow {
        id: string;
        name: string;
        value: number | null;
      }
      const nullableData: NullableRow[] = [
        { id: "1", name: "Alpha", value: null },
        { id: "2", name: "Beta", value: 20 },
        { id: "3", name: "Gamma", value: 10 },
      ];
      const nullableColumns: ColumnDef<NullableRow>[] = [
        { id: "name", label: "Name", accessor: (r) => r.name, sortValue: (r) => r.name },
        {
          id: "value",
          label: "Value",
          accessor: (r) => r.value ?? "",
          sortValue: (r) => r.value ?? "",
        },
      ];

      render(
        <ConfigurableTable
          tableName="sort-null-asc"
          columns={nullableColumns}
          data={nullableData}
          rowKey={(r) => r.id}
        />
      );

      // Ascending: nulls at end
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["10", "20", ""]);

      // Descending: nulls still at end
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["20", "10", ""]);
    });

    it("does not mutate original data array", () => {
      const originalData = [...sortTestData];
      const frozenData = Object.freeze([...sortTestData]) as TestRow[];

      render(
        <ConfigurableTable
          tableName="sort-no-mutate"
          columns={testColumns}
          data={frozenData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      // Original data unchanged
      expect(sortTestData.map((d) => d.name)).toEqual(originalData.map((d) => d.name));
    });

    it("resets to ascending when clicking a different column", () => {
      render(
        <ConfigurableTable
          tableName="sort-reset-col"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      // Sort Name descending
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Charlie", "Beta", "Alpha"]);

      // Click Value → should sort ascending
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["10", "20", "30"]);
    });

    it("produces no duplicate rows after sorting", () => {
      render(
        <ConfigurableTable
          tableName="sort-no-dup"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );

      // Before sort: 3 data rows
      const rowsBefore = screen.getAllByRole("row").length - 1; // minus header
      expect(rowsBefore).toBe(3);

      // Sort ascending
      fireEvent.click(screen.getByText("Name"));
      const rowsAsc = screen.getAllByRole("row").length - 1;
      expect(rowsAsc).toBe(3);

      // Sort descending
      fireEvent.click(screen.getByText("Name"));
      const rowsDesc = screen.getAllByRole("row").length - 1;
      expect(rowsDesc).toBe(3);

      // Clear sort
      fireEvent.click(screen.getByText("Name"));
      const rowsClear = screen.getAllByRole("row").length - 1;
      expect(rowsClear).toBe(3);
    });

    it("shows correct sort indicator (arrow)", () => {
      const { container } = render(
        <ConfigurableTable
          tableName="sort-indicator"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      // No indicator initially
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeNull();
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeNull();

      // Ascending indicator
      fireEvent.click(screen.getByText("Name"));
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeTruthy();
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeNull();

      // Descending indicator
      fireEvent.click(screen.getByText("Name"));
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeNull();
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeTruthy();

      // Cleared - no indicator
      fireEvent.click(screen.getByText("Name"));
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeNull();
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeNull();
    });

    it("does not sort columns without sortValue", () => {
      const noSortColumns: ColumnDef<TestRow>[] = [
        { id: "name", label: "Name", accessor: (r) => r.name },
        { id: "value", label: "Value", accessor: (r) => r.value, sortValue: (r) => r.value },
      ];
      render(
        <ConfigurableTable
          tableName="no-sort"
          columns={noSortColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Charlie", "Alpha", "Beta"]);
    });

    it("sort works when data changes (e.g. filtered data)", () => {
      const { rerender } = render(
        <ConfigurableTable
          tableName="sort-filter"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );

      // Sort ascending by name
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Alpha", "Beta", "Charlie"]);

      // Simulate filtered data (remove Charlie)
      const filteredData = sortTestData.filter((r) => r.name !== "Charlie");
      rerender(
        <ConfigurableTable
          tableName="sort-filter"
          columns={testColumns}
          data={filteredData}
          rowKey={(r) => r.id}
        />
      );

      // Sort should still apply to new data
      expect(getCellTexts(0)).toEqual(["Alpha", "Beta"]);
      // No duplicates
      expect(screen.getAllByRole("row").length - 1).toBe(2);
    });

    it("sort persists when data is replaced (simulating page change)", () => {
      const page1: TestRow[] = [
        { id: "1", name: "Charlie", value: 30, extra: "a" },
        { id: "2", name: "Alpha", value: 10, extra: "b" },
      ];
      const page2: TestRow[] = [
        { id: "3", name: "Zeta", value: 5, extra: "c" },
        { id: "4", name: "Delta", value: 15, extra: "d" },
      ];

      const { rerender } = render(
        <ConfigurableTable
          tableName="sort-paginate"
          columns={testColumns}
          data={page1}
          rowKey={(r) => r.id}
        />
      );

      // Sort by name ascending
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Alpha", "Charlie"]);

      // Switch to page 2
      rerender(
        <ConfigurableTable
          tableName="sort-paginate"
          columns={testColumns}
          data={page2}
          rowKey={(r) => r.id}
        />
      );

      // Sort should still apply
      expect(getCellTexts(0)).toEqual(["Delta", "Zeta"]);
    });
  });
});
