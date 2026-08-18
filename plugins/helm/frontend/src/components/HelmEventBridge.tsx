import { FC } from "react";
import { useHelmCleanupEvents } from "../hooks/async-events/useHelmCleanupEvents";
import { useHelmInstallEvents } from "../hooks/async-events/useHelmInstallEvents";
import { useHelmUpgradeEvents } from "../hooks/async-events/useHelmUpgradeEvents";
import { usePluginBackendRestarted } from "../hooks/async-events/usePluginBackendRestarted";

/**
 * Mounted by the host as soon as the plugin is READY (regardless of which
 * view is active), so install/upgrade/cleanup completion events raised by
 * backend operations started from anywhere in the Helm UI are always caught.
 */
export const HelmEventBridge: FC = () => {
  useHelmInstallEvents();
  useHelmUpgradeEvents();
  useHelmCleanupEvents();
  usePluginBackendRestarted();
  return null;
};
