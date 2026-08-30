import { useInfiniteQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../api/api";
import { QUERY_KEY_HELM_REPOSITORY_CATALOG } from "../../api/api.const";
import type { HelmRepositoryCatalogPage } from "../../api/resources";
import { SearchHelmRepositoryCatalog } from "../../api/resources";

const PAGE_SIZE = 20;

export const useSearchHelmRepositoryCatalog = (query: string) =>
  useInfiniteQuery<HelmRepositoryCatalogPage, Error>({
    queryKey: [QUERY_KEY_HELM_REPOSITORY_CATALOG, query],
    queryFn: ({ pageParam }) => SearchHelmRepositoryCatalog(query, pageParam as number, PAGE_SIZE),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.HasMore
        ? allPages.reduce((total, page) => total + page.Entries.length, 0)
        : undefined,
    ...DEFAULT_QUERY_OPTIONS,
  });
