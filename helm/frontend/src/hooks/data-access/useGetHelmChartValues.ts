import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_CHART_VALUES } from "../../api/api.const";
import { GetHelmChartValues } from "../../api/resources";

export function useGetHelmChartValues(repo: string, chartName: string, version: string) {
  return useQuery({
    queryKey: [QUERY_KEY_HELM_CHART_VALUES, repo, chartName, version],
    queryFn: () => GetHelmChartValues(repo, chartName, version),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!repo && !!chartName && !!version,
  });
}
