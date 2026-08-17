import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import type { HelmRelease } from "../../api/resources";
import { ListHelmReleases } from "../../api/resources";

export const useGetHelmReleases = (context: string, namespace: string) =>
  useQuery<HelmRelease[], Error>({
    queryKey: [QUERY_KEY_HELM_RELEASES, context, namespace],
    queryFn: () => ListHelmReleases(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
