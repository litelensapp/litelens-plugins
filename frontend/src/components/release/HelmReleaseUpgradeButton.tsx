import {
  ArrowUpIcon,
  Button,
  DropdownMenuItem,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@litelens/design-system";
import { FC } from "react";

export interface HelmReleaseUpgradeButtonProps {
  disabled?: boolean;
  onClick: () => void;
  mode?: "menu-item" | "icon-button";
}

export const HelmReleaseUpgradeButton: FC<HelmReleaseUpgradeButtonProps> = ({
  disabled,
  onClick,
  mode = "icon-button",
}) => {
  if (mode === "menu-item") {
    return (
      <DropdownMenuItem disabled={disabled} onClick={onClick}>
        <ArrowUpIcon className="mr-2 size-3.5" />
        Upgrade
      </DropdownMenuItem>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={disabled}
            onClick={onClick}
            aria-label="Upgrade Helm Release"
          >
            <ArrowUpIcon className="size-4" />
          </Button>
        }
      />
      <TooltipContent side="bottom" sideOffset={4}>
        Upgrade Helm Release
      </TooltipContent>
    </Tooltip>
  );
};
