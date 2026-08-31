import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { QUERY_KEY_HELM_REPOSITORIES } from "../../api/api.const";
import { queryClient } from "../../api/query.client";
import { AddHelmRepository } from "../../api/resources";
import { getErrorMessage } from "../../utils";

export const useAddHelmRepository = () => {
  return useMutation({
    mutationFn: ({ name, url }: { name: string; url: string }) => AddHelmRepository(name, url),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_REPOSITORIES] });
      renderSuccessToast({
        title: "Repository added",
        description: `${name} has been added to your repositories`,
      });
    },
    onError: (err, { name }) =>
      renderErrorToast({
        title: "Failed to add repository",
        description: `${name}: ${getErrorMessage(err)}`,
      }),
  });
};
