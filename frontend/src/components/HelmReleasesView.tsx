import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  PackageIcon,
  ResourceDeletionButton,
  ResourceLink,
  RocketIcon,
  SearchInput,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useHelmContext } from "../HelmContext";
import { useGetHelmReleases } from "../hooks/data-access/useGetHelmReleases";
import {
  useDeleteHelmRelease,
  useDeleteHelmReleaseWithCleanup,
} from "../hooks/data-mutation/useDeleteHelmRelease";
import { decodeValuesYAML } from "../utils";
import { HelmReleaseCleanupConfirmationModal } from "./HelmReleaseCleanupConfirmationModal";
import { HelmReleaseDeleteConfirmationModal } from "./HelmReleaseDeleteConfirmationModal";
import { HelmReleaseDetailDrawer } from "./HelmReleaseDetailDrawer";
import { HelmReleaseRollbackButton } from "./HelmReleaseRollbackButton";
import { HelmReleaseRollbackModal } from "./HelmReleaseRollbackModal";
import { HelmReleaseStatusBadge } from "./HelmReleaseStatusBadge";
import { HelmReleaseUpgradeButton } from "./HelmReleaseUpgradeButton";

interface HelmReleaseTableCtaButtonsProps {
  name: string;
  namespace: string;
  chart: string;
  chartVersion: string;
  repository: string;
  valuesYAML: string;
  revision: number;
}

const HelmReleaseTableCtaButtons: FC<HelmReleaseTableCtaButtonsProps> = ({
  name,
  namespace,
  chart,
  chartVersion,
  repository,
  valuesYAML,
  revision,
}) => {
  const { activeContext, unifiedTray } = useHelmContext();

  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [cleanupModalOpen, setCleanupModalOpen] = useState(false);
  const [rollbackModalOpen, setRollbackModalOpen] = useState(false);
  const [isDecoding, setIsDecoding] = useState(false);

  const { mutate: deleteRelease, isPending } = useDeleteHelmRelease();
  const { mutate: deleteReleaseWithCleanup, isPending: isCleanupPending } =
    useDeleteHelmReleaseWithCleanup();

  const isAnyMutationPending = isPending || isCleanupPending || isDecoding;

  const handleUpgradeClick = async () => {
    setIsDecoding(true);
    try {
      const currentValuesYAML = await decodeValuesYAML(valuesYAML);
      unifiedTray?.openTab("helm-chart-upgrade", {
        label: `Helm Upgrade: ${name}`,
        icon: <PackageIcon className="size-3.5 shrink-0" />,
        dedupeKey: `${chart}/${namespace}/${name}`,
        chartName: chart,
        currentVersion: chartVersion,
        namespace,
        releaseName: name,
        currentValuesYAML,
        activeContext,
        repository,
      });
    } finally {
      setIsDecoding(false);
    }
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="Actions"
          className="hover:bg-accent flex size-6 cursor-pointer items-center justify-center rounded-sm"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreVerticalIcon className="size-3.5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-fit">
          <HelmReleaseUpgradeButton
            mode="menu-item"
            disabled={isAnyMutationPending}
            onClick={handleUpgradeClick}
          />
          <HelmReleaseRollbackButton
            disabled={isAnyMutationPending}
            onClick={() => setRollbackModalOpen(true)}
          />
          <ResourceDeletionButton onClick={() => setDeleteModalOpen(true)} />
          <ResourceDeletionButton
            label="Delete & Cleanup"
            className="text-destructive focus:text-destructive"
            onClick={() => setCleanupModalOpen(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <HelmReleaseDeleteConfirmationModal
        open={deleteModalOpen}
        name={name}
        namespace={namespace}
        isPending={isPending}
        onClose={() => setDeleteModalOpen(false)}
        onConfirm={() => {
          deleteRelease(
            { namespace, releaseName: name },
            { onSuccess: () => setDeleteModalOpen(false) }
          );
        }}
      />

      <HelmReleaseCleanupConfirmationModal
        open={cleanupModalOpen}
        name={name}
        namespace={namespace}
        isPending={isCleanupPending}
        onClose={() => setCleanupModalOpen(false)}
        onConfirm={() => {
          deleteReleaseWithCleanup(
            { namespace, releaseName: name },
            { onSuccess: () => setCleanupModalOpen(false) }
          );
        }}
      />

      <HelmReleaseRollbackModal
        open={rollbackModalOpen}
        namespace={namespace}
        releaseName={name}
        currentRevision={revision}
        onClose={() => setRollbackModalOpen(false)}
      />
    </>
  );
};

export const HelmReleasesView: FC = () => {
  const {
    activeContext,
    namespace,
    selectedHelmReleaseName,
    selectedHelmReleaseNamespace,
    onToggleHelmReleaseDetail,
    onToggleNamespaceDetail,
  } = useHelmContext();

  const [search, setSearch] = useState("");

  const { data: raw = [], isLoading } = useGetHelmReleases(activeContext, namespace);

  const releases = raw
    .filter(
      (r) =>
        !search ||
        r.Name.toLowerCase().includes(search.toLowerCase()) ||
        r.Namespace.toLowerCase().includes(search.toLowerCase()) ||
        r.Chart.toLowerCase().includes(search.toLowerCase())
    )
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Releases</span>
        <span className="text-muted-foreground text-xs">
          {releases.length} item{releases.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <SearchInput
            placeholder="Search Releases..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            wrapperClassName="w-68"
          />
        </div>
      </div>

      <Table containerClassName="flex-1 overflow-y-auto">
        <TableHeader className="bg-background z-sticky sticky top-0">
          <TableRow>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Chart</TableHead>
            <TableHead>Chart Version</TableHead>
            <TableHead>App Version</TableHead>
            <TableHead>Revision</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Updated</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespace ? 7 : 8}
              includeCheckbox={false}
              columnWidths={[
                "w-[65%]",
                "w-[55%]",
                "w-[45%]",
                "w-[35%]",
                "w-[30%]",
                "w-[35%]",
                "w-[35%]",
                "w-[35%]",
              ]}
            />
          ) : releases.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespace ? 8 : 9} className="px-0 py-0">
                <EmptyState
                  icon={<RocketIcon className="size-8" />}
                  title="No Helm Releases"
                  description="Install a chart to create a release"
                />
              </TableCell>
            </TableRow>
          ) : (
            releases.map((rel) => (
              <TableRow
                key={`${rel.Namespace}/${rel.Name}`}
                onClick={() => onToggleHelmReleaseDetail(rel.Namespace, rel.Name)}
                className="hover:bg-muted/50 cursor-pointer"
              >
                <TableCell className="font-mono text-xs">{rel.Name}</TableCell>
                {!namespace && (
                  <TableCell className="text-xs">
                    <ResourceLink
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleNamespaceDetail(rel.Namespace);
                      }}
                    >
                      {rel.Namespace}
                    </ResourceLink>
                  </TableCell>
                )}
                <TableCell className="font-mono text-xs">{rel.Chart}</TableCell>
                <TableCell className="font-mono text-xs">{rel.ChartVersion || "—"}</TableCell>
                <TableCell className="font-mono text-xs">{rel.AppVersion || "—"}</TableCell>
                <TableCell className="font-mono text-xs">{rel.Revision}</TableCell>
                <TableCell>
                  <HelmReleaseStatusBadge status={rel.Status} />
                </TableCell>
                <TableCell className="text-muted-foreground text-xs">{rel.Updated}</TableCell>
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <HelmReleaseTableCtaButtons
                    name={rel.Name}
                    namespace={rel.Namespace}
                    chart={rel.Chart}
                    chartVersion={rel.ChartVersion}
                    repository={rel.Repository}
                    valuesYAML={rel.EncodedValuesYAML}
                    revision={rel.Revision}
                  />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <HelmReleaseDetailDrawer
        releaseName={selectedHelmReleaseName}
        namespace={selectedHelmReleaseNamespace}
        open={!!selectedHelmReleaseName}
        onClose={onToggleHelmReleaseDetail}
      />
    </div>
  );
};
