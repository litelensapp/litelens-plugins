import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  invalidateBackendAddrCache,
  ListHelmCharts,
  ListHelmRepositories,
  ListHelmReleases,
  PluginError,
} from "../wailsBridge";

// Mock window.go.app.App and global fetch
const mockGetPluginBackendAddr = vi.fn();
const mockFetch = vi.fn();

beforeEach(() => {
  // Set up global window mocks
  (global as any).window = {
    go: {
      app: {
        App: {
          GetPluginBackendAddr: mockGetPluginBackendAddr,
        },
      },
    },
  };

  (global as any).fetch = mockFetch;
  invalidateBackendAddrCache();
});

afterEach(() => {
  vi.clearAllMocks();
  invalidateBackendAddrCache();
});

describe("wailsBridge", () => {
  describe("successful fetch returns parsed data", () => {
    it("ListHelmCharts fetches and parses data successfully", async () => {
      const mockData = [
        { name: "chart1", repository: "repo1" },
        { name: "chart2", repository: "repo2" },
      ];

      mockGetPluginBackendAddr.mockResolvedValue("127.0.0.1:5000");
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockData,
      });

      const result = await ListHelmCharts();

      expect(result).toEqual(mockData);
      expect(mockGetPluginBackendAddr).toHaveBeenCalledWith("helm");
      expect(mockFetch).toHaveBeenCalledWith("http://127.0.0.1:5000/api/helm/listCharts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
    });

    it("ListHelmRepositories fetches and parses data successfully", async () => {
      const mockData = [{ name: "repo1", url: "https://example.com" }];

      mockGetPluginBackendAddr.mockResolvedValue("127.0.0.1:5000");
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockData,
      });

      const result = await ListHelmRepositories();

      expect(result).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalledWith("http://127.0.0.1:5000/api/helm/listRepositories", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
    });
  });

  describe("backend error response throws without retry", () => {
    it("throws PluginError when backend returns error response", async () => {
      const errorPayload: PluginError = {
        code: "INVALID_REQUEST",
        message: "Invalid namespace",
      };

      mockGetPluginBackendAddr.mockResolvedValue("127.0.0.1:5000");
      mockFetch.mockResolvedValue({
        ok: false,
        json: async () => errorPayload,
      });

      await expect(ListHelmReleases("invalid-ns")).rejects.toEqual(errorPayload);
      // Verify GetPluginBackendAddr was called only once (no retry)
      expect(mockGetPluginBackendAddr).toHaveBeenCalledTimes(1);
    });
  });

  describe("fetch TypeError triggers one refetch + retry", () => {
    it("retries once and succeeds on second attempt after TypeError", async () => {
      const mockData = [{ name: "release1" }];

      mockGetPluginBackendAddr
        .mockResolvedValueOnce("127.0.0.1:5000")
        .mockResolvedValueOnce("127.0.0.1:5001"); // Fresh address on retry

      mockFetch
        .mockRejectedValueOnce(new TypeError("Network request failed"))
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockData,
        });

      const result = await ListHelmReleases("default");

      expect(result).toEqual(mockData);
      // Verify GetPluginBackendAddr was called twice (initial + refetch)
      expect(mockGetPluginBackendAddr).toHaveBeenCalledTimes(2);
      // Verify fetch was called twice
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe("both attempts fail with TypeError throws PLUGIN_UNAVAILABLE", () => {
    it("throws PLUGIN_UNAVAILABLE when both attempts fail with TypeError", async () => {
      mockGetPluginBackendAddr
        .mockResolvedValueOnce("127.0.0.1:5000")
        .mockResolvedValueOnce("127.0.0.1:5001");

      mockFetch.mockRejectedValue(new TypeError("Network request failed"));

      await expect(ListHelmCharts()).rejects.toEqual({
        code: "PLUGIN_UNAVAILABLE",
        message: "Plugin backend unreachable",
      });

      // Verify both GetPluginBackendAddr calls happened
      expect(mockGetPluginBackendAddr).toHaveBeenCalledTimes(2);
      // Verify both fetch attempts happened
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe("address caching", () => {
    it("uses cached address for subsequent calls", async () => {
      const mockData1 = [{ name: "chart1" }];
      const mockData2 = [{ name: "repo1" }];

      mockGetPluginBackendAddr.mockResolvedValue("127.0.0.1:5000");
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockData1,
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockData2,
        });

      await ListHelmCharts();
      await ListHelmRepositories();

      // GetPluginBackendAddr should be called only once (cached)
      expect(mockGetPluginBackendAddr).toHaveBeenCalledTimes(1);
      // Both fetch calls should happen
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it("invalidateBackendAddrCache clears cached address", async () => {
      const mockData = [{ name: "chart1" }];

      mockGetPluginBackendAddr.mockResolvedValue("127.0.0.1:5000");
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockData,
      });

      await ListHelmCharts();
      invalidateBackendAddrCache();
      await ListHelmCharts();

      // GetPluginBackendAddr should be called twice (cache was invalidated)
      expect(mockGetPluginBackendAddr).toHaveBeenCalledTimes(2);
    });
  });
});
