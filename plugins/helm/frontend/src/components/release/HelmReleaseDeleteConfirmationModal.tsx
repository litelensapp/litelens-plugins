import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface HelmReleaseDeleteConfirmationModalProps {
  open: boolean;
  name: string;
  namespace: string;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const HelmReleaseDeleteConfirmationModal: FC<HelmReleaseDeleteConfirmationModalProps> = ({
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
        Uninstall Helm Release:{" "}
        <span className="font-mono font-normal text-muted-foreground">{name}</span>
      </>
    }
    description={
      <>
        This will permanently uninstall{" "}
        <span className="font-mono font-medium text-foreground">{name}</span> from namespace{" "}
        <span className="font-mono font-medium text-foreground">{namespace}</span>. All associated
        resources will be removed.
      </>
    }
    confirmLabel="Uninstall"
    confirmVariant="destructive"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);
