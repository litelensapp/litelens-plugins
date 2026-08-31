import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { QUERY_KEY_HELM_REPOSITORIES } from "../../api/api.const";
import { queryClient } from "../../api/query.client";
import { RemoveHelmRepository } from "../../api/resources";

export const useRemoveHelmRepository = () => {
  return useMutation({
    mutationFn: (name: string) => RemoveHelmRepository(name),
    onSuccess: (_, name) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_REPOSITORIES] });
      renderSuccessToast({
        title: "Repository removed",
        description: `${name} has been removed from your repositories`,
      });
    },
    onError: (err, name) =>
      renderErrorToast({
        title: "Failed to remove repository",
        description: `${name}: ${String(err)}`,
      }),
  });
};
