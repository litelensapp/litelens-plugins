import { DropdownMenuItem, RotateCcwIcon } from "@litelens/design-system";
import { FC } from "react";

export interface HelmReleaseRollbackButtonProps {
  disabled?: boolean;
  onClick: () => void;
}

export const HelmReleaseRollbackButton: FC<HelmReleaseRollbackButtonProps> = ({
  disabled,
  onClick,
}) => (
  <DropdownMenuItem disabled={disabled} onClick={onClick}>
    <RotateCcwIcon className="mr-2 size-3.5" />
    Rollback
  </DropdownMenuItem>
);
