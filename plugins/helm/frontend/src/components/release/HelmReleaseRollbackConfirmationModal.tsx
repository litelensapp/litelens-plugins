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
        <span className="font-mono font-normal text-muted-foreground">{releaseName}</span>
      </>
    }
    description={
      <>
        This will roll back{" "}
        <span className="font-mono font-medium text-foreground">{releaseName}</span> in namespace{" "}
        <span className="font-mono font-medium text-foreground">{namespace}</span> to revision{" "}
        <span className="font-mono font-medium text-foreground">{targetRevision}</span>.
      </>
    }
    confirmLabel="Rollback"
    confirmVariant="default"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);
