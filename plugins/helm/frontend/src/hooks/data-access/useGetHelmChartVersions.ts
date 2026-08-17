import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_CHART_VERSIONS } from "../../api/api.const";
import { ListHelmChartVersions } from "../../api/resources";

export function useGetHelmChartVersions(repo: string, chartName: string) {
  return useQuery({
    queryKey: [QUERY_KEY_HELM_CHART_VERSIONS, repo, chartName],
    queryFn: () => ListHelmChartVersions(repo, chartName),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!repo && !!chartName,
  });
}
