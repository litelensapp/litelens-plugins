import {
  Button,
  FullTextSearchInput,
  Input,
  Loader2Icon,
  RotateCcwIcon,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  cn,
  useFullTextSearch,
} from "@litelens/design-system";
import { FC, Fragment, useReducer, useState } from "react";
import { useHelmContext } from "../../HelmContext";
import { useGetHelmChartValues } from "../../hooks/data-access/useGetHelmChartValues";
import { useGetHelmChartVersions } from "../../hooks/data-access/useGetHelmChartVersions";
import { useInstallHelmChart } from "../../hooks/data-mutation/useInstallHelmChart";
import { HelmChartVersionSelectDropdown } from "./HelmChartVersionSelectDropdown";

export interface HelmChartVersionTrayTab {
  id: string;
  repo: string;
  chartName: string;
  initialVersion: string;
  activeContext: string;
  onNavigateToView: (view: string) => void;
}

// ---- TrayToolbar ----

interface TrayToolbarProps {
  collapsed: boolean;
  repo: string;
  chartName: string;
  versions: string[];
  selectedVersion: string;
  onVersionSelect: (v: string) => void;
  namespaces: Array<{ Name: string }>;
  namespace: string;
  onNamespaceChange: (v: string) => void;
  releaseName: string;
  onReleaseNameChange: (v: string) => void;
  isLoading: boolean;
  isInstalling: boolean;
  onCancel: () => void;
  onInstall: () => void;
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
  namespaces,
  namespace,
  onNamespaceChange,
  releaseName,
  onReleaseNameChange,
  isLoading,
  isInstalling,
  onCancel,
  onInstall,
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
      {repo}/{chartName}
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
      disabled={isLoading || isInstalling}
    />

    <div className="bg-border h-4 w-px" />

    {/* Namespace */}
    <span className="text-muted-foreground text-xs">Namespace</span>
    <Select
      value={namespace}
      onValueChange={(value) => {
        if (value) onNamespaceChange(value);
      }}
      disabled={isLoading || isInstalling}
    >
      <SelectTrigger className="h-7 w-36 text-xs" aria-label="Namespace">
        <SelectValue placeholder="Select namespace" />
      </SelectTrigger>
      <SelectContent
        className="w-fit"
        positionerClassName="z-popover-nested"
        alignItemWithTrigger={false}
      >
        {namespaces.map((ns) => (
          <SelectItem key={ns.Name} value={ns.Name} className="text-xs">
            {ns.Name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>

    {/* Name */}
    <Input
      placeholder="Name (optional)"
      value={releaseName}
      onChange={(e) => onReleaseNameChange(e.target.value)}
      className="h-7 w-40 text-xs"
      aria-label="Release name"
      disabled={isLoading || isInstalling}
    />

    <div className="flex-1" />

    {/* SearchIcon */}
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
      disabled={isLoading || isInstalling}
    >
      Cancel
    </Button>
    <Button
      size="sm"
      onClick={onInstall}
      disabled={isLoading || isInstalling || !selectedVersion}
      className="h-7 text-xs"
      aria-label="Install"
    >
      {isInstalling ? <Loader2Icon className="size-3.5 animate-spin" /> : "Install"}
    </Button>
  </div>
);

// ---- values editor state ----

interface ValuesEditorState {
  editedValues: string;
  prevValues: string | undefined;
}

type ValuesEditorAction =
  { type: "sync_from_source"; values: string | undefined } | { type: "edit"; value: string };

function valuesEditorReducer(
  state: ValuesEditorState,
  action: ValuesEditorAction
): ValuesEditorState {
  switch (action.type) {
    case "sync_from_source":
      return { editedValues: action.values ?? "", prevValues: action.values };
    case "edit":
      return { ...state, editedValues: action.value };
  }
}

// ---- HelmChartVersionTray ----

export interface HelmChartVersionTrayProps {
  tab: HelmChartVersionTrayTab;
  collapsed: boolean;
  onClose: () => void;
}

export const HelmChartVersionTray: FC<HelmChartVersionTrayProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  const { repo, chartName, initialVersion, onNavigateToView } = tab;

  const { namespaces = [] } = useHelmContext();

  const [releaseName, setReleaseName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [selectedVersion, setSelectedVersion] = useState(initialVersion);
  const [{ editedValues, prevValues }, dispatchValuesEditor] = useReducer(valuesEditorReducer, {
    editedValues: "",
    prevValues: undefined,
  });

  const { data: versions = [] } = useGetHelmChartVersions(repo, chartName);
  const {
    data: values,
    isLoading: isValuesLoading,
    isError,
    error,
    isFetching: isValuesFetching,
  } = useGetHelmChartValues(repo, chartName, selectedVersion);

  // Reset editedValues when values changes (initial load + version switch) — derived state pattern
  if (values !== prevValues) {
    dispatchValuesEditor({ type: "sync_from_source", values });
  }

  const isLoading = isValuesLoading || isValuesFetching;
  const isDirty = editedValues !== (values ?? "");

  const { mutate, isPending: isInstalling } = useInstallHelmChart({
    onNavigateToReleases: () => onNavigateToView("helm-releases"),
  });

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
    dispatchValuesEditor({ type: "edit", value: values ?? "" });
  };

  const handleInstall = () => {
    mutate(
      {
        namespace,
        releaseName: releaseName?.trim() || undefined,
        repository: repo,
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
        repo={repo}
        chartName={chartName}
        versions={versions}
        selectedVersion={selectedVersion}
        onVersionSelect={setSelectedVersion}
        namespaces={namespaces}
        namespace={namespace}
        onNamespaceChange={setNamespace}
        releaseName={releaseName}
        onReleaseNameChange={setReleaseName}
        isLoading={isLoading}
        isInstalling={isInstalling}
        onCancel={onClose}
        onInstall={handleInstall}
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
        {isLoading ? (
          <div className="flex flex-col gap-2 p-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <Fragment key={i}>
                <div className="bg-muted h-4 w-full animate-pulse rounded" />
              </Fragment>
            ))}
          </div>
        ) : isError ? (
          <p className="text-destructive p-4 text-xs">Failed to load values: {String(error)}</p>
        ) : !values ? (
          <p className="text-muted-foreground p-4 text-xs">
            No values.yaml available for this chart version.
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
                      aria-label="Reset values to defaults"
                    >
                      <RotateCcwIcon className="size-3.5" />
                    </Button>
                  }
                />
                <TooltipContent side="bottom" sideOffset={4}>
                  Reset to default values
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
