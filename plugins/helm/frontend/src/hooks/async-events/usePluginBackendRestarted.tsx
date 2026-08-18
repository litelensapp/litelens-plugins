import { useEffect } from "react";
import { invalidateBackendAddrCache } from "../../api/wailsBridge";
import { EventsOn } from "../../wailsRuntimeBridge";

interface PluginBackendRestartedPayload {
  pluginID: string;
  backendAddr: string;
}

export const usePluginBackendRestarted = () => {
  useEffect(() => {
    const unsub = EventsOn("plugin:backendRestarted", (data: unknown) => {
      const payload = data as PluginBackendRestartedPayload;
      if (payload.pluginID === "helm") {
        invalidateBackendAddrCache();
      }
    });
    return unsub;
  }, []);
};
