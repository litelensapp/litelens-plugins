import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { RollbackHelmRelease } from "../../api/resources";

export const useRollbackHelmRelease = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      namespace,
      releaseName,
      revision,
    }: {
      namespace: string;
      releaseName: string;
      revision: number;
    }) => RollbackHelmRelease(namespace, releaseName, revision),
    onSuccess: (_, { releaseName }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      renderSuccessToast({
        title: "Helm release rolled back",
        description: `${releaseName} has been rolled back successfully`,
      });
    },
    onError: (err, { releaseName }) =>
      renderErrorToast({
        title: "Failed to rollback Helm release",
        description: `${releaseName}: ${String(err)}`,
      }),
  });
};
