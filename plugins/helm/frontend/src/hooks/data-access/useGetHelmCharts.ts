import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_CHARTS } from "../../api/api.const";
import type { HelmChart } from "../../api/resources";
import { ListHelmCharts } from "../../api/resources";

export const useGetHelmCharts = () =>
  useQuery<HelmChart[], Error>({
    queryKey: [QUERY_KEY_HELM_CHARTS],
    queryFn: () => ListHelmCharts(),
    ...DEFAULT_QUERY_OPTIONS,
  });
