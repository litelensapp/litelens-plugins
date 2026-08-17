import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface HelmReleaseCleanupConfirmationModalProps {
  open: boolean;
  name: string;
  namespace: string;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const HelmReleaseCleanupConfirmationModal: FC<HelmReleaseCleanupConfirmationModalProps> = ({
  open,
  name,
  namespace,
  isPending,
  onClose,
  onConfirm,
}) => (
  <ConfirmationModal
    open={open}
    title={
      <>
        Delete & Cleanup:{" "}
        <span className="text-muted-foreground font-mono font-normal">{name}</span>
      </>
    }
    description={
      <>
        This will uninstall <span className="text-foreground font-mono font-medium">{name}</span>{" "}
        from namespace <span className="text-foreground font-mono font-medium">{namespace}</span>{" "}
        and delete all associated resources (pods, PVCs, PVs, config maps, secrets, etc.). This
        action cannot be undone.
      </>
    }
    confirmLabel="Delete & Cleanup"
    confirmVariant="destructive"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);
