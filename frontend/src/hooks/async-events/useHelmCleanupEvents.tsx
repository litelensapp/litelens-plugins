import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useQueryClient } from "@tanstack/react-query";
import { EventsOn } from "../../wailsRuntimeBridge";
import { useEffect } from "react";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";

interface CleanupCompletePayload {
  releaseName: string;
  deleted: number;
}

interface CleanupPartialPayload {
  releaseName: string;
  deleted: number;
  errors: string[];
}

interface CleanupErrorPayload {
  releaseName: string;
  error: string;
}

export const useHelmCleanupEvents = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const offComplete = EventsOn("helm:cleanup:complete", (payload: unknown) => {
      const { releaseName, deleted } = payload as CleanupCompletePayload;
      const description =
        deleted > 0
          ? `${deleted} resource${deleted === 1 ? "" : "s"} cleaned up for ${releaseName}`
          : `No leftover resources found for ${releaseName}`;
      renderSuccessToast({ title: "Resources cleaned up", description: description });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
    });

    const offPartial = EventsOn("helm:cleanup:partial", (payload: unknown) => {
      const { releaseName, deleted, errors } = payload as CleanupPartialPayload;
      renderSuccessToast({
        title: "Cleanup partially complete",
        description: `${deleted} resource${deleted === 1 ? "" : "s"} removed for ${releaseName}; ${errors.length} failed to delete`,
      });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
    });

    const offError = EventsOn("helm:cleanup:error", (payload: unknown) => {
      const { releaseName, error } = payload as CleanupErrorPayload;
      renderErrorToast({
        title: "Resource cleanup failed",
        description: `Could not clean up resources for ${releaseName}: ${error}`,
      });
    });

    return () => {
      offComplete();
      offPartial();
      offError();
    };
  }, [queryClient]);
};
