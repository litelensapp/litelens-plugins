import {
  Button,
  FullTextSearchInput,
  Loader2Icon,
  RotateCcwIcon,
  Textarea,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  cn,
  useFullTextSearch,
} from "@litelens/design-system";
import { FC, useReducer, useState } from "react";
import { useGetHelmChartVersions } from "../hooks/data-access/useGetHelmChartVersions";
import { useUpgradeHelmChart } from "../hooks/data-mutation/useUpgradeHelmChart";
import { HelmChartVersionSelectDropdown } from "./HelmChartVersionSelectDropdown";

export interface HelmChartVersionUpgradeTrayTab {
  id: string;
  chartName: string;
  currentVersion: string;
  namespace: string;
  releaseName: string;
  currentValuesYAML: string;
  activeContext: string;
  repository: string;
}

// ---- TrayToolbar ----

interface TrayToolbarProps {
  collapsed: boolean;
  repo: string;
  chartName: string;
  versions: string[];
  selectedVersion: string;
  onVersionSelect: (v: string) => void;
  namespace: string;
  releaseName: string;
  isUpgrading: boolean;
  onCancel: () => void;
  onUpgrade: () => void;
  searchTerm: string;
  matchCount: number;
  currentMatchIdx: number;
  onSearch: (term: string) => void;
  onSearchNext: () => void;
}

const TrayToolbar: FC<TrayToolbarProps> = ({
  collapsed,
  repo,
  chartName,
  versions,
  selectedVersion,
  onVersionSelect,
  namespace,
  releaseName,
  isUpgrading,
  onCancel,
  onUpgrade,
  searchTerm,
  matchCount,
  currentMatchIdx,
  onSearch,
  onSearchNext,
}) => (
  <div className={cn("flex h-10 shrink-0 items-center gap-3 border-b px-4", collapsed && "hidden")}>
    {/* Chart chip */}
    <span className="text-muted-foreground text-xs">Chart</span>
    <span className="bg-muted text-foreground rounded px-2 py-0.5 font-mono text-xs">
      {`${repo}/${chartName}`}
    </span>

    <div className="bg-border h-4 w-px" />

    {/* Version */}
    <span className="text-muted-foreground text-xs">Version</span>
    <HelmChartVersionSelectDropdown
      versions={versions}
      selectedVersion={selectedVersion}
      onVersionSelect={onVersionSelect}
      align="start"
      positionerClassName="z-popover-nested"
      className="w-28"
      disabled={isUpgrading}
    />

    <div className="bg-border h-4 w-px" />

    {/* Namespace (read-only) */}
    <span className="text-muted-foreground text-xs">Namespace</span>
    <span className="bg-muted text-foreground rounded px-2 py-0.5 font-mono text-xs">
      {namespace}
    </span>

    <div className="bg-border h-4 w-px" />

    {/* Release name (read-only) */}
    <span className="text-muted-foreground text-xs">Release</span>
    <span className="bg-muted text-foreground rounded px-2 py-0.5 font-mono text-xs">
      {releaseName}
    </span>

    <div className="flex-1" />

    {/* Search */}
    <FullTextSearchInput
      searchTerm={searchTerm}
      matchCount={matchCount}
      currentMatchIdx={currentMatchIdx}
      onSearch={onSearch}
      onSearchNext={onSearchNext}
      ariaLabel="Search YAML"
    />

    {/* Actions */}
    <Button
      variant="ghost"
      size="sm"
      onClick={onCancel}
      className="h-7 text-xs"
      disabled={isUpgrading}
    >
      Cancel
    </Button>
    <Button
      size="sm"
      onClick={onUpgrade}
      disabled={isUpgrading || !selectedVersion}
      className="h-7 text-xs"
      aria-label="Upgrade"
    >
      {isUpgrading ? <Loader2Icon className="size-3.5 animate-spin" /> : "Upgrade"}
    </Button>
  </div>
);

// ---- values editor state ----

interface ValuesEditorState {
  editedValues: string;
  prevValues: string;
}

type ValuesEditorAction =
  { type: "sync_from_source"; values: string } | { type: "edit"; value: string };

function valuesEditorReducer(
  state: ValuesEditorState,
  action: ValuesEditorAction
): ValuesEditorState {
  switch (action.type) {
    case "sync_from_source":
      return { editedValues: action.values, prevValues: action.values };
    case "edit":
      return { ...state, editedValues: action.value };
  }
}

// ---- HelmChartVersionUpgradeTray ----

export interface HelmChartVersionUpgradeTrayProps {
  tab: HelmChartVersionUpgradeTrayTab;
  collapsed: boolean;
  onClose: () => void;
}

export const HelmChartVersionUpgradeTray: FC<HelmChartVersionUpgradeTrayProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  const { chartName, currentVersion, namespace, releaseName, currentValuesYAML, repository } = tab;

  const [selectedVersion, setSelectedVersion] = useState(currentVersion);
  const [{ editedValues }, dispatchValuesEditor] = useReducer(valuesEditorReducer, {
    editedValues: currentValuesYAML,
    prevValues: currentValuesYAML,
  });

  // Repository is now provided by the tab, resolved on the backend during GetHelmReleaseByName
  const { data: versions = [] } = useGetHelmChartVersions(repository, chartName);

  const isDirty = editedValues !== currentValuesYAML;

  const { mutate, isPending: isUpgrading } = useUpgradeHelmChart();

  const {
    searchTerm,
    matchCount,
    currentMatchIdx,
    activeMatchCharIdx,
    contentRef,
    handleSearch,
    handleSearchNext,
  } = useFullTextSearch({ text: editedValues });

  const handleResetValues = () => {
    dispatchValuesEditor({ type: "sync_from_source", values: currentValuesYAML });
  };

  const handleUpgrade = () => {
    if (!repository) {
      return;
    }
    mutate(
      {
        namespace,
        releaseName,
        repository,
        chartName,
        version: selectedVersion,
        valuesYAML: editedValues,
      },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <>
      {/* [B] Toolbar row */}
      <TrayToolbar
        collapsed={collapsed}
        repo={repository}
        chartName={chartName}
        versions={versions}
        selectedVersion={selectedVersion}
        onVersionSelect={setSelectedVersion}
        namespace={namespace}
        releaseName={releaseName}
        isUpgrading={isUpgrading}
        onCancel={onClose}
        onUpgrade={handleUpgrade}
        searchTerm={searchTerm}
        matchCount={matchCount}
        currentMatchIdx={currentMatchIdx}
        onSearch={handleSearch}
        onSearchNext={handleSearchNext}
      />

      {/* [C] Content area */}
      <div
        ref={contentRef}
        className={cn("flex flex-1 flex-col overflow-hidden", collapsed && "hidden")}
      >
        {!currentValuesYAML ? (
          <p className="text-muted-foreground p-4 text-xs">
            No values.yaml available for this release.
          </p>
        ) : (
          <>
            {/* Values header with reset button */}
            <div className="flex h-9 shrink-0 items-center justify-between border-b px-4">
              <span className="text-muted-foreground text-xs">Values</span>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleResetValues}
                      disabled={!isDirty}
                      className="h-7 w-7"
                      aria-label="Reset values to current"
                    >
                      <RotateCcwIcon className="size-3.5" />
                    </Button>
                  }
                />
                <TooltipContent side="bottom" sideOffset={4}>
                  Reset to current values
                </TooltipContent>
              </Tooltip>
            </div>
            {/* Editable textarea */}
            <Textarea
              variant="yaml"
              borderRounded={false}
              className="flex-1"
              value={editedValues}
              onChange={(e) => dispatchValuesEditor({ type: "edit", value: e.target.value })}
              editable={true}
              searchTerm={searchTerm}
              activeMatchCharIdx={activeMatchCharIdx}
            />
          </>
        )}
      </div>
    </>
  );
};
