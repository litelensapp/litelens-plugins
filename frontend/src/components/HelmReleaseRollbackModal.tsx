import {
  FormModal,
  Loader2Icon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useGetHelmReleaseHistory } from "../hooks/data-access/useGetHelmReleaseHistory";
import { useRollbackHelmRelease } from "../hooks/data-mutation/useRollbackHelmRelease";
import { HelmReleaseRollbackConfirmationModal } from "./HelmReleaseRollbackConfirmationModal";
import { HelmReleaseStatusBadge } from "./HelmReleaseStatusBadge";

interface HelmReleaseRollbackModalProps {
  open: boolean;
  namespace: string;
  releaseName: string;
  currentRevision: number;
  onClose: () => void;
}

export const HelmReleaseRollbackModal: FC<HelmReleaseRollbackModalProps> = ({
  open,
  namespace,
  releaseName,
  currentRevision,
  onClose,
}) => {
  const [selectedRevision, setSelectedRevision] = useState<number | null>(null);
  const [targetRevision, setTargetRevision] = useState<number | null>(null);
  const [prevOpen, setPrevOpen] = useState(open);

  const { data: revisions = [], isLoading } = useGetHelmReleaseHistory(
    namespace,
    releaseName,
    open
  );

  const { mutate: rollback, isPending: isRollbackPending } = useRollbackHelmRelease();

  // Reset selection when the modal transitions closed, without an Effect (React-recommended pattern).
  if (open !== prevOpen) {
    setPrevOpen(open);
    if (!open) {
      setSelectedRevision(null);
    }
  }

  // Default the selection to the current revision once history loads, without an Effect.
  if (selectedRevision === null && revisions.some((r) => r.Revision === currentRevision)) {
    setSelectedRevision(currentRevision);
  }

  const handleSubmit = async () => {
    if (selectedRevision !== null) {
      setTargetRevision(selectedRevision);
    }
  };

  return (
    <>
      <FormModal
        open={open}
        onClose={onClose}
        title={`Rollback Helm Release: ${releaseName}`}
        submitLabel="Rollback"
        cancelLabel="Cancel"
        size="lg"
        submitDisabled={selectedRevision === null}
        onSubmit={handleSubmit}
      >
        {isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2Icon className="size-4 animate-spin" />
          </div>
        ) : revisions.length === 0 ? (
          <div className="text-muted-foreground py-4 text-sm">No revision history available</div>
        ) : (
          <div className="overflow-hidden rounded-md border">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead>Revision</TableHead>
                  <TableHead>Chart Version</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {revisions.map((rev) => (
                  <TableRow
                    key={rev.Revision}
                    className="hover:bg-muted/50 cursor-pointer"
                    onClick={() => setSelectedRevision(rev.Revision)}
                  >
                    <TableCell>
                      <input
                        type="radio"
                        name="revision"
                        value={String(rev.Revision)}
                        checked={selectedRevision === rev.Revision}
                        onChange={(e) => setSelectedRevision(parseInt(e.target.value))}
                        className="mr-2"
                      />
                      <span className="font-mono text-xs">
                        {rev.Revision}
                        {rev.Revision === currentRevision && (
                          <span className="text-muted-foreground"> (current)</span>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{rev.ChartVersion || "—"}</TableCell>
                    <TableCell>
                      {rev.Status ? <HelmReleaseStatusBadge status={rev.Status} /> : "—"}
                    </TableCell>
                    <TableCell className="text-xs">
                      {rev.UpdatedAt ? new Date(rev.UpdatedAt).toLocaleString() : "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </FormModal>

      <HelmReleaseRollbackConfirmationModal
        open={targetRevision !== null}
        namespace={namespace}
        releaseName={releaseName}
        targetRevision={targetRevision ?? 0}
        isPending={isRollbackPending}
        onClose={() => setTargetRevision(null)}
        onConfirm={() => {
          if (targetRevision === null) return;
          rollback(
            { namespace, releaseName, revision: targetRevision },
            {
              onSuccess: () => {
                setTargetRevision(null);
                onClose();
              },
            }
          );
        }}
      />
    </>
  );
};
