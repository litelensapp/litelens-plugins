import { vi, describe, it, expect, afterEach } from "vitest";
import { render, cleanup } from "@testing-library/react";
import { HelmEventBridge } from "../HelmEventBridge";

vi.mock("../../hooks/async-events/useHelmInstallEvents", () => ({
  useHelmInstallEvents: vi.fn(),
}));

vi.mock("../../hooks/async-events/useHelmUpgradeEvents", () => ({
  useHelmUpgradeEvents: vi.fn(),
}));

vi.mock("../../hooks/async-events/useHelmCleanupEvents", () => ({
  useHelmCleanupEvents: vi.fn(),
}));

vi.mock("../../hooks/async-events/usePluginBackendRestarted", () => ({
  usePluginBackendRestarted: vi.fn(),
}));

describe("HelmEventBridge", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("wires up all async-event hooks", async () => {
    const { useHelmInstallEvents } = await import("../../hooks/async-events/useHelmInstallEvents");
    const { useHelmUpgradeEvents } = await import("../../hooks/async-events/useHelmUpgradeEvents");
    const { useHelmCleanupEvents } = await import("../../hooks/async-events/useHelmCleanupEvents");
    const { usePluginBackendRestarted } =
      await import("../../hooks/async-events/usePluginBackendRestarted");

    render(<HelmEventBridge />);

    expect(useHelmInstallEvents).toHaveBeenCalledTimes(1);
    expect(useHelmUpgradeEvents).toHaveBeenCalledTimes(1);
    expect(useHelmCleanupEvents).toHaveBeenCalledTimes(1);
    expect(usePluginBackendRestarted).toHaveBeenCalledTimes(1);
  });

  it("renders nothing", () => {
    const { container } = render(<HelmEventBridge />);
    expect(container.firstChild).toBeNull();
  });
});
