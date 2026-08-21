import {
  ChevronDownIcon,
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  PackageIcon,
  SearchInput,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
  TruncatedText,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useHelmContext } from "../../HelmContext";
import { useGetHelmCharts } from "../../hooks/data-access/useGetHelmCharts";
import { useGetHelmRepositories } from "../../hooks/data-access/useGetHelmRepositories";
import { HelmChartDetailDrawer } from "./HelmChartDetailDrawer";
import { HelmChartIcon } from "./HelmChartIcon";

export const HelmChartsView: FC = () => {
  const { selectedHelmChartName, selectedHelmChartRepo, onToggleHelmChartDetail } =
    useHelmContext();

  const [search, setSearch] = useState("");
  const [selectedRepos, setSelectedRepos] = useState<Set<string>>(new Set());

  const { data: raw = [], isLoading, isError, error } = useGetHelmCharts();

  const { data: reposData = [] } = useGetHelmRepositories();
  const repos = reposData.map((r) => r.Name);

  const charts = raw
    .filter(
      (c) =>
        (!search ||
          c.Name.toLowerCase().includes(search.toLowerCase()) ||
          c.Description.toLowerCase().includes(search.toLowerCase())) &&
        (selectedRepos.size === 0 || selectedRepos.has(c.Repository))
    )
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  function toggleRepo(repo: string) {
    setSelectedRepos((prev) => {
      const next = new Set(prev);
      if (next.has(repo)) next.delete(repo);
      else next.add(repo);
      return next;
    });
  }

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Charts</span>
        <span className="text-xs text-muted-foreground">
          {charts.length} item{charts.length === 1 ? "" : "s"}
        </span>

        <div className="ml-auto flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger
              className="flex h-8 w-48 cursor-pointer items-center justify-between gap-1.5 rounded-md border border-input px-3 text-xs hover:bg-accent"
              disabled={isLoading || !repos.length}
            >
              {selectedRepos.size === 0
                ? "Repositories"
                : `${selectedRepos.size} repo${selectedRepos.size === 1 ? "" : "s"}`}
              <ChevronDownIcon className="size-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-fit overflow-y-auto">
              {repos.map((repo) => (
                <DropdownMenuCheckboxItem
                  key={repo}
                  checked={selectedRepos.has(repo)}
                  onCheckedChange={() => toggleRepo(repo)}
                >
                  {repo}
                </DropdownMenuCheckboxItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <SearchInput
            placeholder="Search Charts..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            disabled={isLoading}
            wrapperClassName="w-64"
          />
        </div>
      </div>

      <Table containerClassName="flex-1 overflow-y-auto">
        <TableHeader className="z-sticky sticky top-0 bg-background">
          <TableRow>
            <TableHead>Repository</TableHead>
            <TableHead className="w-64">Name</TableHead>
            <TableHead>Description</TableHead>
            <TableHead>Latest Version</TableHead>
            <TableHead>App Version</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && (
            <TableSkeletonLoader
              rows={5}
              columns={5}
              includeCheckbox={false}
              columnWidths={["w-[45%]", "w-[55%]", "w-[65%]", "w-[30%]", "w-[30%]"]}
            />
          )}
          {!isLoading && isError && (
            <TableRow>
              <TableCell colSpan={5} className="px-0 py-0">
                <EmptyState
                  icon={<PackageIcon className="size-8" />}
                  title="Failed to Load Charts"
                  description={
                    error?.message ||
                    "Unable to fetch Helm charts. Check plugin status or try again."
                  }
                />
              </TableCell>
            </TableRow>
          )}
          {!isLoading && !isError && !charts?.length && (
            <TableRow>
              <TableCell colSpan={5} className="px-0 py-0">
                <EmptyState
                  icon={<PackageIcon className="size-8" />}
                  title="No Helm Charts"
                  description="Add a repository to browse available charts"
                />
              </TableCell>
            </TableRow>
          )}
          {!isLoading &&
            !!charts?.length &&
            charts.map((chart) => (
              <TableRow
                key={`${chart.Repository}/${chart.Name}`}
                className="cursor-pointer"
                onClick={() => onToggleHelmChartDetail(chart.Repository, chart.Name)}
              >
                <TableCell className="text-xs">{chart.Repository}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <HelmChartIcon src={chart.Icon} size="size-4" />
                    <span className="font-mono text-xs">{chart.Name}</span>
                  </div>
                </TableCell>
                <TableCell className="max-w-80 text-xs text-muted-foreground">
                  <TruncatedText text={chart.Description || "—"} />
                </TableCell>
                <TableCell className="font-mono text-xs">{chart.Version}</TableCell>
                <TableCell className="font-mono text-xs">{chart.AppVersion || "—"}</TableCell>
              </TableRow>
            ))}
        </TableBody>
      </Table>

      <HelmChartDetailDrawer
        chartName={selectedHelmChartName}
        repository={selectedHelmChartRepo}
        open={!!selectedHelmChartName}
        onClose={onToggleHelmChartDetail}
      />
    </div>
  );
};
