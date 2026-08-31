import { Button, Loader2Icon, ScrollArea, Trash2Icon } from "@litelens/design-system";
import { useGetHelmRepositories } from "../../hooks/data-access/useGetHelmRepositories";
import { useRemoveHelmRepository } from "../../hooks/data-mutation/useRemoveHelmRepository";
import { HelmRepositoriesSelect } from "./HelmRepositoriesSelect";

export const HelmSettingsTab = () => {
  const { data: configuredRepos = [] } = useGetHelmRepositories();
  const {
    mutate: removeRepo,
    isPending: isRemoving,
    variables: removingName,
  } = useRemoveHelmRepository();

  const configuredNames = new Set(configuredRepos.map((r) => r.Name));

  return (
    <div className="flex h-full max-w-lg flex-col gap-6">
      <div className="flex flex-col gap-2">
        <label
          htmlFor="add-repository-trigger"
          className="text-left text-xs font-semibold tracking-wider uppercase"
        >
          Add Repository
        </label>
        <HelmRepositoriesSelect
          triggerId="add-repository-trigger"
          configuredNames={configuredNames}
        />
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-2">
        <span className="text-left text-xs font-semibold tracking-wider uppercase">
          Configured Repositories
        </span>
        <ScrollArea className="min-h-0 rounded-md border border-input">
          <div className="flex flex-col divide-y divide-border">
            {!configuredRepos.length && (
              <div className="p-4 text-center text-xs text-muted-foreground">
                No repositories configured yet.
              </div>
            )}
            {configuredRepos.map((r) => {
              const isThisRemoving = isRemoving && removingName === r.Name;
              return (
                <div key={r.Name} className="flex items-center justify-between gap-2 px-3 py-2">
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate text-xs font-medium">{r.Name}</span>
                    <span className="truncate text-xs text-muted-foreground">{r.URL}</span>
                  </div>
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    disabled={isThisRemoving}
                    onClick={() => removeRepo(r.Name)}
                  >
                    {isThisRemoving ? (
                      <Loader2Icon className="size-3.5 animate-spin" />
                    ) : (
                      <Trash2Icon className="size-3.5 text-destructive" />
                    )}
                  </Button>
                </div>
              );
            })}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
};
