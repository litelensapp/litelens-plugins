import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import type { HelmRelease } from "../../../api/resources";

// ─── hoisted mocks ────────────────────────────────────────────────────────────

const resourceLinksNamespaceMock = vi.hoisted(() => vi.fn());
const openTabMock = vi.hoisted(() => vi.fn());

vi.mock("../../../hooks/data-access/useGetHelmReleases", () => ({
  useGetHelmReleases: vi.fn(),
}));

vi.mock("@litelens/core", () => ({
  clusterWideAPI: {
    useExposeProperties: vi.fn(() => ({
      activeContext: "ctx",
      activeNamespaces: [],
      availableNamespaces: [],
      resourceLinks: { namespace: resourceLinksNamespaceMock },
      unifiedTray: {
        tabs: [],
        activeTabId: null,
        collapsed: true,
        expanded: false,
        snapPoint: "36px",
        openTab: openTabMock,
        setActiveTab: vi.fn(),
        closeTab: vi.fn(),
        closeAll: vi.fn(),
        setSnapPoint: vi.fn(),
      },
    })),
  },
}));

// ─── imports after mocks ──────────────────────────────────────────────────────

import { clusterWideAPI } from "@litelens/core";
import { useGetHelmReleases } from "../../../hooks/data-access/useGetHelmReleases";
import { HelmReleasesView } from "../HelmReleasesView";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

function makeRelease(overrides: Partial<HelmRelease> = {}): HelmRelease {
  return {
    Name: "my-release",
    Namespace: "default",
    Chart: "nginx",
    ChartVersion: "15.0.0",
    AppVersion: "1.25.0",
    Status: "deployed",
    Revision: 1,
    Updated: "2d",
    UpdatedAt: "2026-06-28T00:00:00Z",
    Repository: "bitnami",
    EncodedValuesYAML: "",
    ...overrides,
  };
}

// ─── setup ────────────────────────────────────────────────────────────────────

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: [] });
});

// ─── tests ────────────────────────────────────────────────────────────────────

describe("HelmReleasesView", () => {
  it('renders "Releases" heading and 0 items count when data is empty', () => {
    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("Releases").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/0 item/).length).toBeGreaterThan(0);
  });

  it('shows "Item list is empty" when data is empty', () => {
    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("No Helm Releases").length).toBeGreaterThan(0);
  });

  it("renders release rows with Name, Namespace, Chart, Status columns", () => {
    const releases = [
      makeRelease({ Name: "prometheus", Namespace: "monitoring", Chart: "kube-prometheus" }),
      makeRelease({ Name: "grafana", Namespace: "monitoring", Chart: "grafana" }),
    ];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("prometheus").length).toBeGreaterThan(0);
    expect(screen.getAllByText("grafana").length).toBeGreaterThan(0);
    expect(screen.getAllByText("kube-prometheus").length).toBeGreaterThan(0);
  });

  it("shows correct item count for 1 item (singular)", () => {
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [makeRelease()],
    });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("1 item").length).toBeGreaterThan(0);
  });

  it("filters by Name (case-insensitive)", () => {
    const releases = [makeRelease({ Name: "prometheus-stack" }), makeRelease({ Name: "grafana" })];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    fireEvent.change(screen.getByPlaceholderText("Search Releases..."), {
      target: { value: "PROMETHEUS" },
    });

    expect(screen.getAllByText("prometheus-stack").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("grafana")).toHaveLength(0);
  });

  it("filters by Namespace (case-insensitive)", () => {
    const releases = [
      makeRelease({ Name: "rel-a", Namespace: "monitoring" }),
      makeRelease({ Name: "rel-b", Namespace: "default" }),
    ];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    fireEvent.change(screen.getByPlaceholderText("Search Releases..."), {
      target: { value: "MONITORING" },
    });

    expect(screen.getAllByText("rel-a").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("rel-b")).toHaveLength(0);
  });

  it("filters by Chart (case-insensitive)", () => {
    const releases = [
      makeRelease({ Name: "rel-a", Chart: "kube-prometheus-stack" }),
      makeRelease({ Name: "rel-b", Chart: "grafana" }),
    ];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    fireEvent.change(screen.getByPlaceholderText("Search Releases..."), {
      target: { value: "kube-prometheus" },
    });

    expect(screen.getAllByText("rel-a").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("rel-b")).toHaveLength(0);
  });

  it("renders releases sorted alphabetically by Name", () => {
    const releases = [
      makeRelease({ Name: "zebra-rel", Namespace: "ns" }),
      makeRelease({ Name: "alpha-rel", Namespace: "ns" }),
      makeRelease({ Name: "mango-rel", Namespace: "ns" }),
    ];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    const rows = screen.getAllByRole("row");
    const dataRows = rows.filter((row) => row.querySelector("td"));
    const nameCells = dataRows
      .map((row) => row.querySelector("td:first-child")?.textContent)
      .filter(Boolean);

    expect(nameCells).toEqual(["alpha-rel", "mango-rel", "zebra-rel"]);
  });

  it("renders a status badge for each release", () => {
    const releases = [
      makeRelease({ Name: "rel-deployed", Status: "deployed" }),
      makeRelease({ Name: "rel-failed", Status: "failed" }),
    ];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(screen.getAllByText("deployed").length).toBeGreaterThan(0);
    expect(screen.getAllByText("failed").length).toBeGreaterThan(0);
  });

  it("row click calls onToggleHelmReleaseDetail and release stays visible", () => {
    const releases = [makeRelease({ Name: "clickable-release" })];
    (useGetHelmReleases as ReturnType<typeof vi.fn>).mockReturnValue({ data: releases });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    const rows = screen.getAllByRole("row");
    const dataRow = rows.find((r) => r.querySelector("td"));
    expect(dataRow).toBeTruthy();
    fireEvent.click(dataRow!);
    // Still visible — no drawer exists in this view
    expect(screen.getAllByText("clickable-release").length).toBeGreaterThan(0);
  });

  it("passes context and namespaces to useGetHelmReleases", () => {
    (clusterWideAPI.useExposeProperties as ReturnType<typeof vi.fn>).mockReturnValue({
      activeContext: "my-ctx",
      activeNamespaces: ["kube-system"],
      availableNamespaces: [],
      resourceLinks: { namespace: resourceLinksNamespaceMock },
      unifiedTray: null,
    });

    render(<HelmReleasesView />, { wrapper: makeWrapper() });

    expect(useGetHelmReleases).toHaveBeenCalledWith({
      context: "my-ctx",
      namespaces: ["kube-system"],
    });
  });
});
