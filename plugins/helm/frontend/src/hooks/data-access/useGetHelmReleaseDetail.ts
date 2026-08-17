import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASE_DETAIL } from "../../api/api.const";
import { GetHelmReleaseByName } from "../../api/resources";

export function useGetHelmReleaseDetail(context: string, namespace: string, releaseName: string) {
  return useQuery({
    queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL, context, namespace, releaseName],
    queryFn: () => GetHelmReleaseByName(namespace, releaseName),
    enabled: !!context && !!namespace && !!releaseName,
  });
}
