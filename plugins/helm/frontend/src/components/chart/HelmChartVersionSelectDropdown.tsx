import {
  CheckIcon,
  ChevronDownIcon,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Loader2Icon,
  cn,
} from "@litelens/design-system";
import { FC } from "react";

interface HelmChartVersionSelectDropdownProps {
  versions: string[];
  selectedVersion: string;
  onVersionSelect: (v: string) => void;
  isLoading?: boolean;
  disabled?: boolean;
  align?: "start" | "end";
  positionerClassName?: string;
  className?: string;
}

export const HelmChartVersionSelectDropdown: FC<HelmChartVersionSelectDropdownProps> = ({
  versions,
  selectedVersion,
  onVersionSelect,
  isLoading = false,
  disabled = false,
  align = "end",
  positionerClassName,
  className,
}) => (
  <DropdownMenu>
    <DropdownMenuTrigger
      disabled={disabled || isLoading || versions.length === 0}
      className={cn(
        "group/button inline-flex h-7 cursor-pointer items-center justify-between gap-1.5 rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 font-mono text-[0.8rem] font-medium whitespace-nowrap transition-all hover:bg-muted hover:text-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/75 disabled:pointer-events-none disabled:opacity-50 aria-expanded:bg-muted aria-expanded:text-foreground",
        className
      )}
    >
      {isLoading ? (
        <Loader2Icon className="mx-auto size-3.5 animate-spin" />
      ) : (
        <>
          {selectedVersion}
          <ChevronDownIcon className="size-3 shrink-0" />
        </>
      )}
    </DropdownMenuTrigger>
    <DropdownMenuContent
      align={align}
      positionerClassName={positionerClassName}
      className="max-h-52 overflow-y-auto"
    >
      {versions.map((v) => (
        <DropdownMenuItem key={v} onClick={() => onVersionSelect(v)} className="font-mono text-xs">
          {v}
          {v === selectedVersion && <CheckIcon className="ml-auto size-3.5 text-green-500" />}
        </DropdownMenuItem>
      ))}
    </DropdownMenuContent>
  </DropdownMenu>
);
