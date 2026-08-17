import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_REPOSITORIES } from "../../api/api.const";
import type { HelmRepository } from "../../api/resources";
import { ListHelmRepositories } from "../../api/resources";

export const useGetHelmRepositories = () =>
  useQuery<HelmRepository[], Error>({
    queryKey: [QUERY_KEY_HELM_REPOSITORIES],
    queryFn: () => ListHelmRepositories(),
    ...DEFAULT_QUERY_OPTIONS,
  });
