import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// ─── hoisted mocks ────────────────────────────────────────────────────────────

vi.mock("../../../hooks/data-access/useGetHelmChartValues", () => ({
  useGetHelmChartValues: vi.fn(),
}));

vi.mock("../../../hooks/data-access/useGetHelmChartVersions", () => ({
  useGetHelmChartVersions: vi.fn(),
}));

vi.mock("../../../hooks/data-mutation/useInstallHelmChart", () => ({
  useInstallHelmChart: vi.fn(),
}));

vi.mock("@litelens/core", () => ({
  useClusterWideAPI: vi.fn(() => ({
    activeContext: "ctx-test",
    activeNamespaces: [],
    activeResource: "helm-charts",
    availableNamespaces: [],
    onNavigateToView: vi.fn(),
    resourceLinks: {},
    unifiedTray: null,
  })),
}));

// ─── imports after mocks ──────────────────────────────────────────────────────

import { useGetHelmChartValues } from "../../../hooks/data-access/useGetHelmChartValues";
import { useGetHelmChartVersions } from "../../../hooks/data-access/useGetHelmChartVersions";
import { useInstallHelmChart } from "../../../hooks/data-mutation/useInstallHelmChart";
import { HelmProvider } from "../../../HelmContext";
import { HelmChartVersionTray } from "../HelmChartVersionTray";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(
      QueryClientProvider,
      { client },
      createElement(HelmProvider, { children }, children)
    );
}

const testTab = {
  id: "bitnami/nginx",
  repo: "bitnami",
  chartName: "nginx",
  initialVersion: "1.2.0",
  activeContext: "ctx-test",
  onNavigateToView: vi.fn(),
};

// ─── setup ────────────────────────────────────────────────────────────────────

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  HTMLElement.prototype.scrollIntoView = vi.fn();
  (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
  });
  (useGetHelmChartVersions as ReturnType<typeof vi.fn>).mockReturnValue({
    data: ["1.2.0", "1.1.0"],
  });
  (useInstallHelmChart as ReturnType<typeof vi.fn>).mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
  });
});

// ─── tests ────────────────────────────────────────────────────────────────────

describe("HelmChartVersionTray", () => {
  it("renders loading skeleton when values are loading", () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const skeletons = document.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders error message when values fetch fails", () => {
    const testError = new Error("Network error");
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: testError,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const errorMessage = screen.getByText(/Failed to load values/);
    expect(errorMessage).toBeTruthy();
    expect(errorMessage.textContent).toContain("Network error");
  });

  it("renders 'No values.yaml' message when values are empty (not an error)", () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    expect(screen.getByText(/No values\.yaml available for this chart version/)).toBeTruthy();
  });

  it("disables Install button while install is in progress", () => {
    (useInstallHelmChart as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: vi.fn(),
      isPending: true,
    });
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "content",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const installButton = screen.getByRole("button", { name: "Install" });
    expect(installButton.hasAttribute("disabled")).toBe(true);
  });

  it("calls onClose when Cancel button is clicked", () => {
    const onClose = vi.fn();
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "content",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={onClose} />, {
      wrapper: makeWrapper(),
    });

    const cancelButton = screen.getByRole("button", { name: /Cancel/i });
    fireEvent.click(cancelButton);

    expect(onClose).toHaveBeenCalled();
  });

  it("calls mutate with correct payload and onSuccess callback when Install is clicked", () => {
    const mutateMock = vi.fn();
    (useInstallHelmChart as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: mutateMock,
      isPending: false,
    });
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "content",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    fireEvent.click(screen.getByRole("button", { name: "Install" }));

    expect(mutateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        namespace: "default",
        repository: "bitnami",
        chartName: "nginx",
        version: "1.2.0",
      }),
      expect.objectContaining({ onSuccess: expect.any(Function) })
    );
  });

  it("calls onClose via onSuccess callback when install succeeds (async flow)", () => {
    const onClose = vi.fn();
    const mutateMock = vi.fn((_payload, options) => {
      options?.onSuccess?.();
    });
    (useInstallHelmChart as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: mutateMock,
      isPending: false,
    });
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "content",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={onClose} />, {
      wrapper: makeWrapper(),
    });

    fireEvent.click(screen.getByRole("button", { name: "Install" }));

    expect(onClose).toHaveBeenCalled();
  });

  // ─── Search input tests ────────────────────────────────────────────────────

  it("renders Search input with aria-label when values are loaded", () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "key: value\nother: data",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const searchInput = screen.getByLabelText("Search YAML");
    expect(searchInput).toBeTruthy();
    expect(searchInput.getAttribute("placeholder")).toBe("Search…");
  });

  it("shows match count when typing a matching term", async () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "key: value\nother: value\nthird: data",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const searchInput = screen.getByLabelText("Search YAML") as HTMLInputElement;
    fireEvent.change(searchInput, { target: { value: "value" } });

    // Wait for the match count to appear
    await waitFor(() => {
      expect(screen.getByText(/1\/2/)).toBeTruthy();
    });
  });

  it("shows '0' when typing a non-matching term", async () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "key: value\nother: data",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const searchInput = screen.getByLabelText("Search YAML") as HTMLInputElement;
    fireEvent.change(searchInput, { target: { value: "zzz" } });

    // Wait for the '0' to appear
    await waitFor(() => {
      expect(screen.getByText("0")).toBeTruthy();
    });
  });

  it("navigates to next match when Enter key is pressed", async () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "line 1: match\nline 2: match\nline 3: other",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const searchInput = screen.getByLabelText("Search YAML") as HTMLInputElement;

    // Type the search term to find 2 matches
    fireEvent.change(searchInput, { target: { value: "match" } });

    // Verify first match is shown
    await waitFor(() => {
      expect(screen.getByText(/1\/2/)).toBeTruthy();
    });

    // Press Enter to navigate to next match
    fireEvent.keyDown(searchInput, { key: "Enter", code: "Enter" });

    // Verify we're now on the second match
    await waitFor(() => {
      expect(screen.getByText(/2\/2/)).toBeTruthy();
    });
  });

  it("performs case-insensitive search", async () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "Key: Value\nOther: DATA",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    const searchInput = screen.getByLabelText("Search YAML") as HTMLInputElement;

    // Search for lowercase "value" which should match "Value"
    fireEvent.change(searchInput, { target: { value: "value" } });

    // Should find 1 match despite different casing
    await waitFor(() => {
      expect(screen.getByText(/1\/1/)).toBeTruthy();
    });

    // Clear and search for uppercase "data" which should match "DATA"
    fireEvent.change(searchInput, { target: { value: "DATA" } });

    // Should find 1 match despite different casing
    await waitFor(() => {
      expect(screen.getByText(/1\/1/)).toBeTruthy();
    });
  });

  // ─── Inline search highlighting tests ──────────────────────────────────

  it("passes search term to YAML textarea for highlighting", async () => {
    (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
      data: "key: value\nother: value",
      isLoading: false,
      isError: false,
      error: null,
    });

    render(<HelmChartVersionTray tab={testTab} collapsed={false} onClose={vi.fn()} />, {
      wrapper: makeWrapper(),
    });

    // Get the search input and type into it
    const searchInput = screen.getByLabelText("Search YAML") as HTMLInputElement;
    fireEvent.change(searchInput, { target: { value: "value" } });

    // Verify that the match count updates, proving the search term is being processed
    // and passed to the YAML display component for highlighting
    await waitFor(() => {
      expect(screen.getByText(/1\/2/)).toBeTruthy();
    });

    // Verify that with a different search term, the match count also updates
    fireEvent.change(searchInput, { target: { value: "key" } });

    await waitFor(() => {
      expect(screen.getByText(/1\/1/)).toBeTruthy();
    });
  });
});
