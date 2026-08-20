import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import type { HelmRelease } from "../../api/resources";
import { ListHelmReleases } from "../../api/resources";
import { filterByNamespaces, getEffectiveNamespace } from "../../utils";

interface UseGetHelmReleasesParams {
  context: string;
  namespaces: string[];
}

export const useGetHelmReleases = ({ context, namespaces }: UseGetHelmReleasesParams) => {
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<HelmRelease[], Error>({
    queryKey: [QUERY_KEY_HELM_RELEASES, context, effectiveNamespace],
    queryFn: async () => filterByNamespaces(await ListHelmReleases(effectiveNamespace), namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};
