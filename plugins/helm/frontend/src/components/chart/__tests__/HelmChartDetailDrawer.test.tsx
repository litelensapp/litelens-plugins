import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";

// ─── hoisted mocks ────────────────────────────────────────────────────────────

vi.mock("../../../hooks/data-access/useGetHelmChartVersions", () => ({
  useGetHelmChartVersions: vi.fn(),
}));

vi.mock("../../../hooks/data-access/useGetHelmChartDetail", () => ({
  useGetHelmChartDetail: vi.fn(),
}));

vi.mock("../../../hooks/data-access/useGetArtifactHubReadme", () => ({
  useGetArtifactHubReadme: vi.fn(),
}));

vi.mock("../../../hooks/data-access/useGetHelmChartValues", () => ({
  useGetHelmChartValues: vi.fn(),
}));

vi.mock("../../../hooks/data-mutation/useInstallHelmChart", () => ({
  useInstallHelmChart: vi.fn(),
}));

vi.mock("@litelens/core", () => ({
  clusterWideAPI: {
    useExposeProperties: vi.fn(),
    useExposeMethods: vi.fn(),
  },
}));

// ─── imports after mocks ──────────────────────────────────────────────────────

import { clusterWideAPI } from "@litelens/core";
import { useGetHelmChartVersions } from "../../../hooks/data-access/useGetHelmChartVersions";
import { useGetHelmChartDetail } from "../../../hooks/data-access/useGetHelmChartDetail";
import { useGetArtifactHubReadme } from "../../../hooks/data-access/useGetArtifactHubReadme";
import { useGetHelmChartValues } from "../../../hooks/data-access/useGetHelmChartValues";
import { useInstallHelmChart } from "../../../hooks/data-mutation/useInstallHelmChart";
import { HelmChartDetailDrawer } from "../HelmChartDetailDrawer";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

// ─── setup ────────────────────────────────────────────────────────────────────

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  (useGetHelmChartVersions as ReturnType<typeof vi.fn>).mockReturnValue({
    data: [],
    isLoading: false,
  });
  (useGetHelmChartDetail as ReturnType<typeof vi.fn>).mockReturnValue({
    data: null,
    isLoading: false,
    isFetching: false,
  });
  (useGetArtifactHubReadme as ReturnType<typeof vi.fn>).mockReturnValue({
    data: undefined,
    isLoading: false,
    isFetching: false,
  });
  (useGetHelmChartValues as ReturnType<typeof vi.fn>).mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
  });
  (useInstallHelmChart as ReturnType<typeof vi.fn>).mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
  });
  vi.mocked(clusterWideAPI.useExposeProperties).mockReturnValue({
    activeContext: "default",
    activeNamespaces: ["default"],
    activeResource: "helm-charts",
    availableNamespaces: [],
    resourceLinks: {},
    unifiedTray: {
      tabs: [],
      activeTabId: null,
      collapsed: true,
      expanded: false,
      snapPoint: "36px",
      openTab: vi.fn(),
      setActiveTab: vi.fn(),
      closeTab: vi.fn(),
      closeAll: vi.fn(),
      setSnapPoint: vi.fn(),
    },
  } as unknown as ReturnType<typeof clusterWideAPI.useExposeProperties>);
  vi.mocked(clusterWideAPI.useExposeMethods).mockReturnValue({
    onNavigateToView: vi.fn(),
  } as unknown as ReturnType<typeof clusterWideAPI.useExposeMethods>);
});

// ─── tests ────────────────────────────────────────────────────────────────────

describe("HelmChartDetailDrawer", () => {
  it("disables Install button when no versions are loaded yet", () => {
    (useGetHelmChartVersions as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: true,
    });
    (useGetHelmChartDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        Name: "nginx",
        Description: "Nginx chart",
        Version: "",
        AppVersion: "1.0",
        Repository: "bitnami",
        Icon: "",
        Home: "",
        Keywords: [],
        Sources: [],
        Maintainers: [],
      },
      isLoading: false,
      isFetching: false,
    });

    render(
      <HelmChartDetailDrawer
        chartName="nginx"
        repository="bitnami"
        open={true}
        onClose={vi.fn()}
      />,
      { wrapper: makeWrapper() }
    );

    const installButtons = screen.getAllByRole("button", { name: /Install/i });
    expect(installButtons[0].hasAttribute("disabled")).toBe(true);
  });
});
