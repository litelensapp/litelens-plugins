import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_CHART_DETAIL } from "../../api/api.const";
import { GetHelmChartDetail } from "../../api/resources";

export function useGetHelmChartDetail(repo: string, chartName: string, version: string) {
  return useQuery({
    queryKey: [QUERY_KEY_HELM_CHART_DETAIL, repo, chartName, version],
    queryFn: () => GetHelmChartDetail(repo, chartName, version),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!repo && !!chartName,
  });
}
