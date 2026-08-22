import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import type { HelmChart } from "../../../api/resources";

// ─── hoisted mocks ────────────────────────────────────────────────────────────

vi.mock("../../../hooks/data-access/useGetHelmCharts", () => ({
  useGetHelmCharts: vi.fn(),
}));

vi.mock("../../../hooks/data-access/useGetHelmRepositories", () => ({
  useGetHelmRepositories: vi.fn(),
}));

vi.mock("../HelmChartDetailDrawer", () => ({
  HelmChartDetailDrawer: () => null,
}));

// ─── imports after mocks ──────────────────────────────────────────────────────

import { useGetHelmCharts } from "../../../hooks/data-access/useGetHelmCharts";
import { useGetHelmRepositories } from "../../../hooks/data-access/useGetHelmRepositories";
import { HelmChartsView } from "../HelmChartsView";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

function makeChart(overrides: Partial<HelmChart> = {}): HelmChart {
  return {
    Name: "test-chart",
    Description: "A test chart",
    Version: "1.0.0",
    AppVersion: "1.0",
    Repository: "stable",
    Icon: "",
    ...overrides,
  };
}

// ─── setup ────────────────────────────────────────────────────────────────────

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: [] });
  (useGetHelmRepositories as ReturnType<typeof vi.fn>).mockReturnValue({ data: [] });
});

// ─── tests ────────────────────────────────────────────────────────────────────

describe("HelmChartsView", () => {
  it('renders "Charts" heading and 0 items count when data is empty', () => {
    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("Charts").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/0 item/).length).toBeGreaterThan(0);
  });

  it('shows "Item list is empty" when data is empty', () => {
    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("No Helm Charts").length).toBeGreaterThan(0);
  });

  it("renders chart rows with Name, Version, Repository columns", () => {
    const charts = [
      makeChart({ Name: "nginx", Version: "15.0.0", Repository: "bitnami" }),
      makeChart({ Name: "redis", Version: "17.0.0", Repository: "bitnami" }),
    ];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("nginx").length).toBeGreaterThan(0);
    expect(screen.getAllByText("redis").length).toBeGreaterThan(0);
    expect(screen.getAllByText("15.0.0").length).toBeGreaterThan(0);
    expect(screen.getAllByText("17.0.0").length).toBeGreaterThan(0);
  });

  it("shows correct item count", () => {
    const charts = [makeChart({ Name: "chart-a" }), makeChart({ Name: "chart-b" })];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText(/2 item/).length).toBeGreaterThan(0);
  });

  it("filters by Name (case-insensitive)", () => {
    const charts = [makeChart({ Name: "nginx-ingress" }), makeChart({ Name: "redis-cluster" })];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    fireEvent.change(screen.getByPlaceholderText("Search Charts..."), {
      target: { value: "NGINX" },
    });

    expect(screen.getAllByText("nginx-ingress").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("redis-cluster")).toHaveLength(0);
  });

  it("filters by Description (case-insensitive)", () => {
    const charts = [
      makeChart({ Name: "chart-a", Description: "Ingress controller" }),
      makeChart({ Name: "chart-b", Description: "Cache backend" }),
    ];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    fireEvent.change(screen.getByPlaceholderText("Search Charts..."), {
      target: { value: "ingress" },
    });

    expect(screen.getAllByText("chart-a").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("chart-b")).toHaveLength(0);
  });

  it("filters by Repository via multi-select dropdown", () => {
    const charts = [
      makeChart({ Name: "chart-a", Repository: "bitnami" }),
      makeChart({ Name: "chart-b", Repository: "stable" }),
    ];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });
    (useGetHelmRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [
        { Name: "bitnami", URL: "https://charts.bitnami.com" },
        { Name: "stable", URL: "https://charts.helm.sh/stable" },
      ],
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    fireEvent.click(screen.getAllByRole("button", { name: /Repositories/i })[0]);
    fireEvent.click(screen.getByRole("menuitemcheckbox", { name: "bitnami" }));

    expect(screen.getAllByText("chart-a").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("chart-b")).toHaveLength(0);
  });

  it("renders charts sorted alphabetically by Name", () => {
    const charts = [
      makeChart({ Name: "zebra", Repository: "r" }),
      makeChart({ Name: "alpha", Repository: "r" }),
      makeChart({ Name: "mango", Repository: "r" }),
    ];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    const rows = screen.getAllByRole("row");
    const dataRows = rows.filter((row) => row.querySelector("td"));
    const nameCells = dataRows
      .map((row) => {
        // Name is rendered inside a span inside a div with an icon; grab the span text
        return row.querySelector("td:nth-child(2) span.font-mono")?.textContent;
      })
      .filter(Boolean);

    expect(nameCells).toEqual(["alpha", "mango", "zebra"]);
  });

  it("does not navigate on row click (no onClick handler on TableRow)", () => {
    const charts = [makeChart({ Name: "clickable-chart" })];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({ data: charts });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    // Clicking a row should not change visible content or throw
    const rows = screen.getAllByRole("row");
    const dataRow = rows.find((r) => r.querySelector("td"));
    expect(dataRow).toBeTruthy();
    fireEvent.click(dataRow!);
    // Still visible — no drawer opened (no drawer exists in this view)
    expect(screen.getAllByText("clickable-chart").length).toBeGreaterThan(0);
  });

  // ─── Error state tests (bug fix verification) ─────────────────────────────────
  // Bug: Previously, error state was dropped from destructuring, so real gRPC failures
  // (e.g. "plugin not ready") were silently treated as "no data", masking actual errors.

  it("shows 'Failed to Load Charts' when useGetHelmCharts returns isError=true with a message", () => {
    const errorMessage = "plugin not ready: helm service unavailable";
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error(errorMessage),
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("Failed to Load Charts").length).toBeGreaterThan(0);
    expect(screen.getAllByText(errorMessage).length).toBeGreaterThan(0);
  });

  it("shows fallback error message when error object has no message", () => {
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: {}, // Error object without message property
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("Failed to Load Charts").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Unable to fetch Helm charts/i).length).toBeGreaterThan(0);
  });

  it("shows 'No Helm Charts' when isError=false and data is empty (genuine empty state, not error)", () => {
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("No Helm Charts").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("Failed to Load Charts")).toHaveLength(0);
  });

  it("shows loading skeleton when isLoading=true (error state ignored during load)", () => {
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    // Should show skeleton loader, not error or empty state
    expect(screen.queryAllByText("Failed to Load Charts")).toHaveLength(0);
    expect(screen.queryAllByText("No Helm Charts")).toHaveLength(0);
  });

  it("shows populated data when isError=false and data has charts", () => {
    const charts = [makeChart({ Name: "working-chart" })];
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: charts,
      isLoading: false,
      isError: false,
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("working-chart").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("Failed to Load Charts")).toHaveLength(0);
    expect(screen.queryAllByText("No Helm Charts")).toHaveLength(0);
  });

  it("prioritizes error state over empty state when both conditions could apply", () => {
    // Edge case: error but data might be [] — error should take precedence
    (useGetHelmCharts as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: false,
      isError: true,
      error: new Error("Fetch failed"),
    });

    render(<HelmChartsView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("Failed to Load Charts").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("No Helm Charts")).toHaveLength(0);
  });
});
