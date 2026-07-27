import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_ARTIFACTHUB_README } from "../../api/api.const";
import { GetArtifactHubReadme } from "../../api/resources";

export function useGetArtifactHubReadme(repo: string, chartName: string, version: string) {
  return useQuery({
    queryKey: [QUERY_KEY_ARTIFACTHUB_README, repo, chartName, version],
    queryFn: async () => {
      const result = await GetArtifactHubReadme(repo, chartName, version);
      return result || null;
    },
    enabled: !!repo && !!chartName,
    retry: false,
    refetchOnWindowFocus: false,
  });
}
