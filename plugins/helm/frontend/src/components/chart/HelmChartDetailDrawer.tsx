import { useClusterWideAPI } from "@litelens/core";
import {
  AnnotationBadge,
  Button,
  Markdown,
  PackageIcon,
  ResourceDetailDrawer,
  ScrollArea,
} from "@litelens/design-system";
import { FC, Fragment, useDeferredValue, useState } from "react";
import { useGetArtifactHubReadme } from "../../hooks/data-access/useGetArtifactHubReadme";
import { useGetHelmChartDetail } from "../../hooks/data-access/useGetHelmChartDetail";
import { useGetHelmChartVersions } from "../../hooks/data-access/useGetHelmChartVersions";
import { HelmChartIcon } from "./HelmChartIcon";
import { HelmChartVersionSelectDropdown } from "./HelmChartVersionSelectDropdown";

const HelmChartMetadata: FC<{
  repo: string;
  chartName: string;
  selectedVersion: string;
  onVersionSelect: (version: string) => void;
  versions: string[];
  isVersionsLoading: boolean;
  onInstallClick?: () => void;
}> = ({
  repo,
  chartName,
  selectedVersion,
  onVersionSelect,
  versions,
  isVersionsLoading,
  onInstallClick,
}) => {
  const {
    data: chart,
    isLoading: isChartLoading,
    isFetching,
  } = useGetHelmChartDetail(repo, chartName, selectedVersion);

  const isLoading = isChartLoading || (isFetching && chart?.Version !== selectedVersion);

  const CtaButtons = (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={onInstallClick}
        disabled={!selectedVersion || isVersionsLoading || isLoading}
      >
        Install
      </Button>
      <HelmChartVersionSelectDropdown
        versions={versions}
        selectedVersion={selectedVersion}
        onVersionSelect={onVersionSelect}
        isLoading={isVersionsLoading}
        disabled={isLoading}
        align="end"
        className="w-32"
      />
    </>
  );

  if (isLoading)
    return (
      <div className="shrink-0 border-b">
        <div className="flex items-start justify-between gap-3 px-4 pt-4 pb-3">
          <div className="flex min-w-0 items-start gap-3">
            <div className="size-10 shrink-0 animate-pulse rounded bg-muted" />
            <div className="flex min-w-0 flex-col gap-1.5 pt-0.5">
              <span className="truncate text-sm leading-tight font-semibold">
                {repo}/{chartName}
              </span>
              <div className="h-3 w-56 animate-pulse rounded bg-muted" />
              <div className="h-3 w-44 animate-pulse rounded bg-muted" />
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-2">{CtaButtons}</div>
        </div>
        <div className="grid grid-cols-[120px_1fr] items-center gap-y-3 px-4 pb-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Fragment key={i}>
              <div className="h-3 w-20 animate-pulse rounded bg-muted" />
              <div className="h-3 w-32 animate-pulse rounded bg-muted" />
            </Fragment>
          ))}
        </div>
      </div>
    );

  if (!chart)
    return (
      <p className="p-4 text-xs text-muted-foreground">No information available for this chart.</p>
    );

  return (
    <div className="shrink-0 border-b">
      <div className="flex items-start justify-between gap-3 px-4 pt-4 pb-3">
        <div className="flex min-w-0 items-start gap-3">
          <HelmChartIcon src={chart.Icon} />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="truncate text-sm leading-tight font-semibold">
              {repo}/{chartName}
            </span>
            <span className="line-clamp-2 text-xs leading-snug text-muted-foreground">
              {chart.Description || "No description"}
            </span>
          </div>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-2">{CtaButtons}</div>
      </div>

      <div className="grid grid-cols-[120px_1fr] items-start gap-y-2 px-4 pb-4 text-xs">
        <span className="text-muted-foreground">App Version</span>
        <span className="font-mono">{chart.AppVersion || "—"}</span>

        <span className="text-muted-foreground">Home</span>
        {chart.Home ? (
          <a
            href={chart.Home}
            target="_blank"
            rel="noreferrer"
            className="truncate font-mono text-info hover:underline"
          >
            {chart.Home}
          </a>
        ) : (
          <span className="font-mono">—</span>
        )}

        <span className="text-muted-foreground">Maintainers</span>
        <span className="font-mono">{chart.Maintainers?.join(", ") || "—"}</span>

        <span className="self-start pt-0.5 text-muted-foreground">Keywords</span>
        <div className="flex flex-wrap gap-1">
          {chart.Keywords?.length ? (
            chart.Keywords.map((kw) => <AnnotationBadge key={kw} label={kw} />)
          ) : (
            <span className="font-mono">—</span>
          )}
        </div>
      </div>
    </div>
  );
};

const HelmChartDescription: FC<{ repo: string; chartName: string; version: string }> = ({
  repo,
  chartName,
  version,
}) => {
  const {
    data: readme,
    isLoading: isInfoLoading,
    isFetching,
    isError,
  } = useGetArtifactHubReadme(repo, chartName, version);

  // Defer markdown rendering to avoid blocking animation frames during expensive parsing.
  // React will render with the previous value first, then update after a time slice.
  const deferredReadme = useDeferredValue(typeof readme === "string" ? readme : "");

  const isLoading = isInfoLoading || isFetching;

  return (
    <ScrollArea className="h-full min-h-0 flex-1">
      {isLoading || (typeof readme === "string" && !deferredReadme) ? (
        <div className="flex flex-col gap-2 p-4">
          <div className="h-4 w-full animate-pulse rounded bg-muted" />
          <div className="h-4 w-5/6 animate-pulse rounded bg-muted" />
          <div className="h-4 w-4/6 animate-pulse rounded bg-muted" />
          <div className="h-4 w-full animate-pulse rounded bg-muted" />
          <div className="h-4 w-3/4 animate-pulse rounded bg-muted" />
        </div>
      ) : isError ? (
        <p className="p-4 text-xs text-destructive">Failed to load documentation.</p>
      ) : deferredReadme ? (
        <Markdown className="px-4 py-3">{deferredReadme}</Markdown>
      ) : (
        <p className="p-4 text-xs text-muted-foreground">
          No documentation available for this chart.
        </p>
      )}
    </ScrollArea>
  );
};

const HelmChartDetailDrawerBody: FC<{ chartName: string; repository: string }> = ({
  chartName,
  repository,
}) => {
  const { activeContext, onNavigateToView, unifiedTray } = useClusterWideAPI();

  const [manualVersion, setManualVersion] = useState("");
  const { data: versions = [], isLoading: isVersionsLoading } = useGetHelmChartVersions(
    repository,
    chartName
  );
  const selectedVersion = versions.includes(manualVersion) ? manualVersion : (versions[0] ?? "");

  return (
    <>
      <HelmChartMetadata
        repo={repository}
        chartName={chartName}
        selectedVersion={selectedVersion}
        onVersionSelect={setManualVersion}
        versions={versions}
        isVersionsLoading={isVersionsLoading}
        onInstallClick={() => {
          unifiedTray?.openTab("helm-chart", {
            label: `Helm Install: ${repository}/${chartName}`,
            icon: <PackageIcon className="size-3.5 shrink-0" />,
            dedupeKey: `${repository}/${chartName}`,
            repo: repository,
            chartName,
            initialVersion: selectedVersion,
            activeContext,
            onNavigateToView,
          });
        }}
      />
      {selectedVersion && (
        <HelmChartDescription repo={repository} chartName={chartName} version={selectedVersion} />
      )}
    </>
  );
};

interface HelmChartDetailDrawerProps {
  chartName: string | null | undefined;
  repository: string | null | undefined;
  open: boolean;
  onClose: () => void;
}

export const HelmChartDetailDrawer: FC<HelmChartDetailDrawerProps> = ({
  chartName,
  repository,
  open,
  onClose,
}) => {
  return (
    <ResourceDetailDrawer open={open} onClose={onClose}>
      {chartName && repository && (
        <HelmChartDetailDrawerBody
          key={`${repository}-${chartName}`}
          chartName={chartName}
          repository={repository}
        />
      )}
    </ResourceDetailDrawer>
  );
};
