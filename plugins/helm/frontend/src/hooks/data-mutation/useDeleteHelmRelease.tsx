import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { DeleteHelmRelease, DeleteHelmReleaseWithCleanup } from "../../api/resources";

export const useDeleteHelmRelease = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ namespace, releaseName }: { namespace: string; releaseName: string }) =>
      DeleteHelmRelease(namespace, releaseName),
    onSuccess: (_, { releaseName }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      renderSuccessToast({
        title: "Helm release deleted",
        description: `${releaseName} has been uninstalled`,
      });
    },
    onError: (err, { releaseName }) =>
      renderErrorToast({
        title: "Failed to delete Helm release",
        description: `${releaseName}: ${String(err)}`,
      }),
  });
};

export const useDeleteHelmReleaseWithCleanup = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ namespace, releaseName }: { namespace: string; releaseName: string }) =>
      DeleteHelmReleaseWithCleanup(namespace, releaseName),
    onSuccess: (_, { releaseName }) => {
      // Go returned — helm uninstall completed. Cleanup runs asynchronously;
      // useHelmCleanupEvents will show the final cleanup toast.
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      renderSuccessToast({
        title: "Helm release deleted",
        description: `${releaseName} has been uninstalled. Other resources are being cleaned up in the background.`,
      });
    },
    onError: (err, { releaseName }) =>
      renderErrorToast({
        title: "Failed to delete Helm release",
        description: `${releaseName}: ${String(err)}`,
      }),
  });
};
