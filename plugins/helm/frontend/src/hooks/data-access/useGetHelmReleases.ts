import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import type { HelmRelease } from "../../api/resources";
import { ListHelmReleases } from "../../api/resources";

interface UseGetHelmReleasesParams {
  context: string;
  namespaces: string[];
}

export const useGetHelmReleases = ({ context, namespaces }: UseGetHelmReleasesParams) => {
  return useQuery<HelmRelease[], Error>({
    // namespaces is no longer sent to the backend (the plugin sources its filter from
    // the host's ActiveNamespacesWatch gRPC push), but stays in the key so a local
    // namespace-selection change still triggers a refetch.
    queryKey: [QUERY_KEY_HELM_RELEASES, context, namespaces],
    queryFn: () => ListHelmReleases(),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};
