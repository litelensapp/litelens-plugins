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
        <span className="font-mono font-normal text-muted-foreground">{name}</span>
      </>
    }
    description={
      <>
        This will uninstall <span className="font-mono font-medium text-foreground">{name}</span>{" "}
        from namespace <span className="font-mono font-medium text-foreground">{namespace}</span>{" "}
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
