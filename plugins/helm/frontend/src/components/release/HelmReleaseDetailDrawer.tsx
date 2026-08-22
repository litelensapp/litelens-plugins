import { clusterWideAPI } from "@litelens/core";
import {
  ButtonGroup,
  FullTextSearchInput,
  LoadingSpinner,
  Markdown,
  PackageIcon,
  ResourceDeletionButton,
  ResourceDetailDrawer,
  ResourceDetailDrawerHeader,
  ResourceDetailEmptyBody,
  ResourceLink,
  ScrollArea,
  SheetTitle,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
  useFullTextSearch,
} from "@litelens/design-system";
import { FC, useEffect, useState } from "react";
import type { HelmReleaseDetail, HelmReleaseResource } from "../../api/resources";
import { useGetHelmReleaseDetail } from "../../hooks/data-access/useGetHelmReleaseDetail";
import {
  useDeleteHelmRelease,
  useDeleteHelmReleaseWithCleanup,
} from "../../hooks/data-mutation/useDeleteHelmRelease";
import { HelmReleaseCleanupConfirmationModal } from "./HelmReleaseCleanupConfirmationModal";
import { HelmReleaseDeleteConfirmationModal } from "./HelmReleaseDeleteConfirmationModal";
import { HelmReleaseStatusBadge } from "./HelmReleaseStatusBadge";
import { HelmReleaseUpgradeButton } from "./HelmReleaseUpgradeButton";

const HelmReleaseOverviewTab: FC<{ data: HelmReleaseDetail }> = ({ data }) => {
  const {
    searchTerm,
    matchCount,
    currentMatchIdx,
    activeMatchCharIdx,
    contentRef,
    handleSearch,
    handleSearchNext,
  } = useFullTextSearch({ text: data.ValuesYAML });

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ScrollArea className="shrink-0">
        <div className="grid grid-cols-[160px_1fr] items-start gap-y-3 p-4">
          <span className="text-h3 text-muted-foreground">Name</span>
          <span className="text-body font-mono">{data.Name}</span>

          <span className="text-h3 text-muted-foreground">Repository</span>
          <span className="text-body font-mono">{data.Repository || "—"}</span>

          <span className="text-h3 text-muted-foreground">Namespace</span>
          <span className="text-body font-mono">{data.Namespace}</span>

          <span className="text-h3 text-muted-foreground">Chart</span>
          <span className="text-body font-mono">{data.Chart}</span>

          <span className="text-h3 text-muted-foreground">Chart Version</span>
          <span className="text-body font-mono">{data.ChartVersion || "—"}</span>

          <span className="text-h3 text-muted-foreground">App Version</span>
          <span className="text-body font-mono">{data.AppVersion || "—"}</span>

          <span className="text-h3 text-muted-foreground">Status</span>
          <HelmReleaseStatusBadge status={data.Status} />

          <span className="text-h3 text-muted-foreground">Revision</span>
          <span className="text-body font-mono">{data.Revision}</span>

          <span className="text-h3 text-muted-foreground">Updated</span>
          <div>
            <span className="text-body">{data.Updated}</span>{" "}
            <span className="font-mono text-muted-foreground">({data.UpdatedAt})</span>
          </div>
        </div>
      </ScrollArea>

      <div ref={contentRef} className="flex min-h-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center justify-between gap-2 border-t bg-muted/50 px-4 py-2 text-xs font-semibold tracking-wide uppercase">
          <span>Values</span>
          <FullTextSearchInput
            searchTerm={searchTerm}
            matchCount={matchCount}
            currentMatchIdx={currentMatchIdx}
            onSearch={handleSearch}
            onSearchNext={handleSearchNext}
            ariaLabel="Search YAML"
          />
        </div>
        <Textarea
          variant="yaml"
          className="mx-3 my-3 min-h-0 flex-1"
          value={data.ValuesYAML}
          searchTerm={searchTerm}
          activeMatchCharIdx={activeMatchCharIdx}
        />
      </div>
    </div>
  );
};

const HelmReleaseResourcesTab: FC<{ resources: HelmReleaseResource[] }> = ({ resources }) => {
  const { resourceLinks } = clusterWideAPI.useExposeProperties();

  if (resources.length === 0) {
    return (
      <p className="p-4 text-xs text-muted-foreground">No resources found for this release.</p>
    );
  }

  const grouped = Object.entries(
    resources.reduce<Record<string, HelmReleaseResource[]>>((acc, r) => {
      if (!acc[r.Kind]) acc[r.Kind] = [];
      acc[r.Kind].push(r);
      return acc;
    }, {})
  ).sort(([a], [b]) => a.localeCompare(b));

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-5 p-4">
        {grouped.map(([kind, items]) => (
          <div key={kind}>
            <p className="mb-1.5 text-xs font-semibold">{kind}</p>
            <div className="overflow-hidden rounded-sm border">
              <table className="w-full table-fixed text-xs">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="w-3/4 px-3 py-2 text-left font-medium text-muted-foreground">
                      Name
                    </th>
                    <th className="w-1/4 px-3 py-2 text-left font-medium text-muted-foreground">
                      Namespace
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((r) => (
                    <tr key={r.Name} className="border-b last:border-0">
                      <td className="px-3 py-2 font-mono">
                        {resourceLinks[kind.toLowerCase()] ? (
                          <ResourceLink
                            truncate
                            onClick={() => resourceLinks[kind.toLowerCase()](r.Namespace, r.Name)}
                          >
                            {r.Name}
                          </ResourceLink>
                        ) : (
                          <span className="block truncate" title={r.Name}>
                            {r.Name}
                          </span>
                        )}
                      </td>
                      <td className="px-3 py-2">
                        {r.Namespace ? (
                          <ResourceLink onClick={() => resourceLinks.namespace("", r.Namespace)}>
                            {r.Namespace}
                          </ResourceLink>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ))}
      </div>
    </ScrollArea>
  );
};

const HelmReleaseNotesTab: FC<{ notes: string }> = ({ notes }) => {
  return (
    <ScrollArea className="h-full">
      <Markdown className="px-4 py-3">{notes}</Markdown>
    </ScrollArea>
  );
};

const HelmReleaseDrawerCtaButtons: FC<{
  data: HelmReleaseDetail;
  name: string;
  namespace: string;
  onDeleted: () => void;
}> = ({ data, name, namespace, onDeleted }) => {
  const { activeContext, unifiedTray } = clusterWideAPI.useExposeProperties();

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showCleanupModal, setShowCleanupModal] = useState(false);

  const { mutate: deleteRelease, isPending: isDeletePending } = useDeleteHelmRelease();
  const { mutate: deleteReleaseWithCleanup, isPending: isCleanupPending } =
    useDeleteHelmReleaseWithCleanup();

  const isAnyMutationPending = isDeletePending || isCleanupPending;

  const handleUpgradeClick = () => {
    // Repository is now resolved from the detail data
    unifiedTray?.openTab("helm-chart-upgrade", {
      label: `Helm Upgrade: ${name}`,
      icon: <PackageIcon className="size-3.5 shrink-0" />,
      dedupeKey: `${data.Chart}/${namespace}/${name}`,
      chartName: data.Chart,
      currentVersion: data.ChartVersion,
      namespace,
      releaseName: name,
      currentValuesYAML: data.ValuesYAML ?? "",
      activeContext,
      repository: data.Repository,
    });
  };

  return (
    <>
      <ButtonGroup>
        <HelmReleaseUpgradeButton disabled={isAnyMutationPending} onClick={handleUpgradeClick} />
        <ResourceDeletionButton
          mode="icon-button"
          ariaLabel="Delete Helm Release"
          disabled={isDeletePending}
          isPending={isDeletePending}
          onClick={() => setShowDeleteModal(true)}
        />
        <ResourceDeletionButton
          mode="icon-button"
          label="Delete & Cleanup"
          ariaLabel="Delete & Cleanup Helm Release"
          disabled={isCleanupPending}
          isPending={isCleanupPending}
          onClick={() => setShowCleanupModal(true)}
          className="text-destructive focus:text-destructive"
        />
      </ButtonGroup>

      <HelmReleaseDeleteConfirmationModal
        open={showDeleteModal}
        name={name}
        namespace={namespace}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={() => {
          deleteRelease({ namespace, releaseName: name }, { onSuccess: onDeleted });
        }}
      />

      <HelmReleaseCleanupConfirmationModal
        open={showCleanupModal}
        name={name}
        namespace={namespace}
        isPending={isCleanupPending}
        onClose={() => setShowCleanupModal(false)}
        onConfirm={() => {
          deleteReleaseWithCleanup({ namespace, releaseName: name }, { onSuccess: onDeleted });
        }}
      />
    </>
  );
};

interface HelmReleaseDetailDrawerProps {
  releaseName: string | null;
  namespace: string | null;
  open: boolean;
  onClose: () => void;
}

const HelmReleaseDetailDrawerBody: FC<
  HelmReleaseDetailDrawerProps & {
    releaseName: string;
    namespace: string;
    onDataChange: (data: HelmReleaseDetail | undefined) => void;
  }
> = ({ releaseName, namespace, onDataChange }) => {
  const { activeContext } = clusterWideAPI.useExposeProperties();
  const { data, isLoading } = useGetHelmReleaseDetail(activeContext, namespace, releaseName);

  useEffect(() => {
    onDataChange(data);
  }, [data, onDataChange]);

  if (isLoading) {
    return <LoadingSpinner className="h-auto flex-1" />;
  }

  if (!data) {
    return <ResourceDetailEmptyBody resourceKind="Helm Release" />;
  }

  return (
    <Tabs defaultValue="overview" className="min-h-0 flex-1">
      <TabsList className="w-full justify-start rounded-none border-b bg-transparent px-4">
        <TabsTrigger value="overview" className="text-xs">
          Overview
        </TabsTrigger>
        <TabsTrigger value="resources" className="text-xs">
          Resources
        </TabsTrigger>
        {data.Notes && (
          <TabsTrigger value="notes" className="text-xs">
            Notes
          </TabsTrigger>
        )}
      </TabsList>

      <TabsContent value="overview" className="mt-0 min-h-0 flex-1 overflow-hidden">
        <HelmReleaseOverviewTab data={data} />
      </TabsContent>

      <TabsContent value="resources" className="mt-0 min-h-0 flex-1 overflow-hidden">
        <HelmReleaseResourcesTab resources={data.Resources ?? []} />
      </TabsContent>

      {data.Notes && (
        <TabsContent value="notes" className="mt-0 min-h-0 flex-1 overflow-auto">
          <HelmReleaseNotesTab notes={data.Notes} />
        </TabsContent>
      )}
    </Tabs>
  );
};

export const HelmReleaseDetailDrawer: FC<HelmReleaseDetailDrawerProps> = ({
  releaseName,
  namespace,
  open,
  onClose,
}) => {
  const { activeContext } = clusterWideAPI.useExposeProperties();

  const hasData = !!activeContext && !!namespace && !!releaseName;

  const [data, setData] = useState<HelmReleaseDetail | undefined>(undefined);

  return (
    <ResourceDetailDrawer open={open} onClose={onClose}>
      <ResourceDetailDrawerHeader>
        <SheetTitle className="text-h1">Helm Release: {releaseName}</SheetTitle>
        {data && (
          <HelmReleaseDrawerCtaButtons
            data={data}
            name={releaseName!}
            namespace={namespace!}
            onDeleted={onClose}
          />
        )}
      </ResourceDetailDrawerHeader>

      {hasData ? (
        <HelmReleaseDetailDrawerBody
          key={`${namespace}/${releaseName}`}
          releaseName={releaseName}
          namespace={namespace}
          open={open}
          onClose={onClose}
          onDataChange={setData}
        />
      ) : (
        <ResourceDetailEmptyBody resourceKind="Helm Release" />
      )}
    </ResourceDetailDrawer>
  );
};
