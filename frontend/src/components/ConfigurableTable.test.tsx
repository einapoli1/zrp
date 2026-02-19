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

const testColumns: ColumnDef<TestRow>[] = [
  { id: "name", label: "Name", accessor: (r) => r.name },
  { id: "value", label: "Value", accessor: (r) => r.value },
  { id: "extra", label: "Extra", accessor: (r) => r.extra, defaultVisible: false },
];

beforeEach(() => {
  localStorage.clear();
});

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
    // Extra column is defaultVisible=false
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
    // "x" and "y" are from the Extra column which is hidden
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
    expect(parsed[2].visible).toBe(false); // extra
  });

  it("restores column state from localStorage", () => {
    // Pre-set state with extra visible and name hidden
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
    // Name column should be hidden
    // We can check that "Alpha" is not shown as table cell (but Name header is gone)
    // Extra "x" should now appear
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
    // Settings button exists (Settings2 icon)
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
});
