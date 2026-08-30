import {
  Input,
  Loader2Icon,
  SearchIcon,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@litelens/design-system";
import { FC, useEffect, useState } from "react";
import { useSearchHelmRepositoryCatalog } from "../../hooks/data-access/useSearchHelmRepositoryCatalog";
import { useAddHelmRepository } from "../../hooks/data-mutation/useAddHelmRepository";

interface HelmRepositoriesSelectProps {
  configuredNames: Set<string>;
}

export const HelmRepositoriesSelect: FC<HelmRepositoriesSelectProps> = ({ configuredNames }) => {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");

  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  const {
    data: catalogPages,
    isLoading: isSearching,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useSearchHelmRepositoryCatalog(debouncedSearch);
  const catalog = catalogPages?.pages.flatMap((page) => page.Entries) ?? [];

  const {
    mutate: addRepo,
    isPending: isAdding,
    variables: addingVariables,
  } = useAddHelmRepository();

  return (
    <Select
      value=""
      onValueChange={(name) => {
        const entry = catalog.find((c) => c.Name === name);
        if (entry && !configuredNames.has(entry.Name)) {
          addRepo({ name: entry.Name, url: entry.URL });
        }
      }}
      onOpenChange={(open) => {
        if (!open) setSearch("");
      }}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Search public repositories..." />
      </SelectTrigger>
      <SelectContent
        alignItemWithTrigger={false}
        className="max-h-72"
        onScroll={(e) => {
          const el = e.currentTarget;
          if (
            hasNextPage &&
            !isFetchingNextPage &&
            el.scrollHeight - el.scrollTop - el.clientHeight < 48
          ) {
            fetchNextPage();
          }
        }}
      >
        <div className="sticky top-0 z-10 border-b bg-popover p-1.5">
          <div className="relative">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              autoFocus
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.stopPropagation()}
              placeholder="Search public repositories..."
              className="pl-8 text-sm"
            />
            {isFetchingNextPage && (
              <Loader2Icon className="pointer-events-none absolute top-1/2 right-2.5 size-3.5 -translate-y-1/2 animate-spin text-muted-foreground" />
            )}
          </div>
        </div>
        {isSearching && (
          <div className="flex items-center justify-center gap-2 p-4 text-xs text-muted-foreground">
            <Loader2Icon className="size-3.5 animate-spin" />
            Searching...
          </div>
        )}
        {!isSearching && !catalog.length && (
          <p className="px-3 py-4 text-center text-xs text-muted-foreground">
            No repositories found.
          </p>
        )}
        {!isSearching &&
          catalog.map((entry) => {
            const alreadyAdded = configuredNames.has(entry.Name);
            const isThisAdding = isAdding && addingVariables?.name === entry.Name;
            return (
              <SelectItem
                key={entry.Name}
                value={entry.Name}
                disabled={alreadyAdded || isThisAdding}
              >
                <div className="flex min-w-0 flex-col">
                  <span className="truncate text-xs font-medium">{entry.Name}</span>
                  <span className="truncate text-xs text-muted-foreground">{entry.URL}</span>
                </div>
              </SelectItem>
            );
          })}
        {!isSearching && isFetchingNextPage && (
          <div className="flex items-center justify-center gap-2 p-2 text-xs text-muted-foreground">
            <Loader2Icon className="size-3.5 animate-spin" />
            Loading more...
          </div>
        )}
      </SelectContent>
    </Select>
  );
};
