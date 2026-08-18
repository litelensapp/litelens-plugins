import { cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import * as wailsBridge from "../../../api/wailsBridge";
import * as wailsRuntimeBridge from "../../../wailsRuntimeBridge";
import { usePluginBackendRestarted } from "../usePluginBackendRestarted";

vi.mock("../../../api/wailsBridge", () => ({
  invalidateBackendAddrCache: vi.fn(),
}));

vi.mock("../../../wailsRuntimeBridge", () => ({
  EventsOn: vi.fn(),
}));

const TestComponent = () => {
  usePluginBackendRestarted();
  return null;
};

describe("usePluginBackendRestarted", () => {
  let mockUnsubscribe: Mock<() => void>;

  beforeEach(() => {
    mockUnsubscribe = vi.fn<() => void>();
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("subscribes to plugin:backendRestarted event on mount", () => {
    const mockEventsOn = vi.mocked(wailsRuntimeBridge.EventsOn);
    mockEventsOn.mockReturnValue(mockUnsubscribe);

    render(<TestComponent />);

    expect(mockEventsOn).toHaveBeenCalledWith("plugin:backendRestarted", expect.any(Function));
  });

  it("calls invalidateBackendAddrCache when the event fires for the helm plugin", () => {
    const mockEventsOn = vi.mocked(wailsRuntimeBridge.EventsOn);
    const mockInvalidate = vi.mocked(wailsBridge.invalidateBackendAddrCache);
    mockEventsOn.mockReturnValue(mockUnsubscribe);

    render(<TestComponent />);
    const callback = mockEventsOn.mock.calls[0][1];
    callback({ pluginID: "helm", backendAddr: "127.0.0.1:5001" });

    expect(mockInvalidate).toHaveBeenCalledTimes(1);
  });

  it("does not call invalidateBackendAddrCache when the event fires for a different plugin", () => {
    const mockEventsOn = vi.mocked(wailsRuntimeBridge.EventsOn);
    const mockInvalidate = vi.mocked(wailsBridge.invalidateBackendAddrCache);
    mockEventsOn.mockReturnValue(mockUnsubscribe);

    render(<TestComponent />);
    const callback = mockEventsOn.mock.calls[0][1];
    callback({ pluginID: "prometheus", backendAddr: "127.0.0.1:5002" });

    expect(mockInvalidate).not.toHaveBeenCalled();
  });

  it("calls the unsubscribe function returned by EventsOn on unmount", () => {
    const mockEventsOn = vi.mocked(wailsRuntimeBridge.EventsOn);
    mockEventsOn.mockReturnValue(mockUnsubscribe);

    const { unmount } = render(<TestComponent />);
    unmount();

    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
  });
});
