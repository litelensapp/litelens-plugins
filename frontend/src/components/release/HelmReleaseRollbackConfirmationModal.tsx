import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface HelmReleaseRollbackConfirmationModalProps {
  open: boolean;
  releaseName: string;
  namespace: string;
  targetRevision: number;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const HelmReleaseRollbackConfirmationModal: FC<
  HelmReleaseRollbackConfirmationModalProps
> = ({ open, releaseName, namespace, targetRevision, isPending, onClose, onConfirm }) => (
  <ConfirmationModal
    open={open}
    title={
      <>
        Rollback Helm Release:{" "}
        <span className="text-muted-foreground font-mono font-normal">{releaseName}</span>
      </>
    }
    description={
      <>
        This will roll back{" "}
        <span className="text-foreground font-mono font-medium">{releaseName}</span> in namespace{" "}
        <span className="text-foreground font-mono font-medium">{namespace}</span> to revision{" "}
        <span className="text-foreground font-mono font-medium">{targetRevision}</span>.
      </>
    }
    confirmLabel="Rollback"
    confirmVariant="default"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);
