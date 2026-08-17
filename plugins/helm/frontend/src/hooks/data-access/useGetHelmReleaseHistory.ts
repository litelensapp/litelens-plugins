import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASE_HISTORY } from "../../api/api.const";
import { GetHelmReleaseHistory } from "../../api/resources";

export const useGetHelmReleaseHistory = (
  namespace: string,
  releaseName: string,
  enabled: boolean = true
) => {
  return useQuery({
    queryKey: [QUERY_KEY_HELM_RELEASE_HISTORY, namespace, releaseName],
    queryFn: () => GetHelmReleaseHistory(namespace, releaseName),
    enabled: enabled && !!namespace && !!releaseName,
  });
};
