import { onError } from "@/utils";
import {
  getResourceClient,
  getResourceListPathFromResource,
  invalidateResourceList,
  Resource,
} from "@/utils/pb";
import {
  Button,
  DefaultMantineColor,
  MantineSize,
  Modal,
  Switch,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  AlertTriangle,
  FileText,
  Loader2,
  ShieldAlert,
  Trash2,
  X,
} from "lucide-react";
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import CopyText from "../CopyText";
import ResourceYAML from "../ResourceYAML";

const DeleteResource = (props: {
  btnSize?:
    | MantineSize
    | "compact-lg"
    | "compact-xs"
    | "compact-sm"
    | "compact-md"
    | "compact-xl";
  btnVariant?: "filled" | "default" | "outline";
  btnColor?: DefaultMantineColor;
  item: Resource;
  onSuccess?: () => void;
  doNotNavigateAfter?: boolean;
  opened?: boolean;
  onClose?: () => void;
  hideTrigger?: boolean;
  btnLabel?: string;
}) => {
  const { item } = props;
  const metadata = item.metadata!;
  const navigate = useNavigate();
  const [isConfirmed, setIsConfirmed] = React.useState(false);
  const [internalOpened, { open, close: closeInternal }] = useDisclosure(false);
  const opened = props.opened ?? internalOpened;
  const close = props.onClose ?? closeInternal;

  const mutationDelete = useMutation({
    mutationFn: async () => {
      // @ts-ignore
      await getResourceClient(item)[`delete${item.kind}`]({
        uid: metadata.uid,
      });
    },
    onSuccess: () => {
      setIsConfirmed(false);
      close();
      invalidateResourceList(item);
      props.onSuccess?.();
      if (!props.doNotNavigateAfter) {
        navigate(getResourceListPathFromResource(item));
      }
      toast.success(`${item.kind} ${metadata.name} deleted`);
    },
    onError,
  });

  React.useEffect(() => {
    if (!opened) {
      setIsConfirmed(false);
      mutationDelete.reset();
    }
  }, [opened, metadata.uid]);

  const handleClose = () => {
    if (mutationDelete.isPending) return;
    setIsConfirmed(false);
    mutationDelete.reset();
    close();
  };

  const errorMessage =
    mutationDelete.error instanceof Error
      ? mutationDelete.error.message
      : "The resource could not be deleted.";

  if (metadata.isSystem) return null;

  return (
    <>
      {!props.hideTrigger && (
        <Button
          type="button"
          color={props.btnColor ?? "red.8"}
          onClick={open}
          size={props.btnSize}
          variant={props.btnVariant}
          leftSection={<Trash2 size={12} strokeWidth={2.25} />}
          aria-label={props.btnLabel ?? `Delete ${item.kind} ${metadata.name}`}
        >
          {props.btnLabel}
        </Button>
      )}

      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        size="md"
        withCloseButton={false}
        padding={0}
        closeOnClickOutside={!mutationDelete.isPending}
        closeOnEscape={!mutationDelete.isPending}
        overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
        transitionProps={{ transition: "pop", duration: 250 }}
        styles={{
          content: {
            border: "1px solid #e2e8f0",
            borderRadius: "14px",
            boxShadow: "0 24px 64px rgba(15,23,42,0.18)",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex flex-col">
          <header className="flex items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
            <div className="flex min-w-0 items-center gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-red-600 text-white shadow-sm">
                <Trash2 size={16} strokeWidth={2.25} />
              </span>
              <div className="min-w-0">
                <h2 className="truncate text-sm font-bold text-slate-900">
                  Delete {item.kind}
                </h2>
                <p className="mt-0.5 text-[0.65rem] font-semibold text-slate-400">
                  Permanent destructive action
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="subtle"
              color="gray"
              size="compact-xs"
              disabled={mutationDelete.isPending}
              leftSection={<X size={12} strokeWidth={2.5} />}
              onClick={handleClose}
            >
              Close
            </Button>
          </header>

          <div className="space-y-4 px-4 py-4 sm:px-5">
            <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50/70 px-3.5 py-3">
              <AlertTriangle
                size={16}
                className="mt-0.5 shrink-0 text-red-600"
                strokeWidth={2.25}
              />
              <div>
                <p className="text-[0.76rem] font-bold text-red-800">
                  This action cannot be undone
                </p>
                <p className="mt-1 text-[0.69rem] font-semibold leading-relaxed text-red-700/80">
                  The resource will be permanently removed from the cluster.
                  Review its identity carefully before continuing.
                </p>
              </div>
            </div>

            <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
              <div className="flex items-start gap-3 border-b border-slate-100 bg-slate-50/60 px-3.5 py-3">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500">
                  <ShieldAlert size={14} strokeWidth={2.25} />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="truncate text-[0.8rem] font-bold text-slate-800">
                      <CopyText value={metadata.name} />
                    </span>
                    <span className="rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[0.58rem] font-bold uppercase tracking-[0.06em] text-slate-500">
                      {item.kind}
                    </span>
                  </div>
                  {metadata.displayName && (
                    <p className="mt-0.5 truncate text-[0.68rem] font-semibold text-slate-400">
                      {metadata.displayName}
                    </p>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-[72px_minmax(0,1fr)] items-center gap-x-3 px-3.5 py-2.5 text-[0.68rem]">
                <span className="font-bold uppercase tracking-[0.06em] text-slate-400">
                  UID
                </span>
                <span className="min-w-0 truncate font-semibold text-slate-500">
                  <CopyText value={metadata.uid} />
                </span>
              </div>
            </section>

            <div className="flex items-center justify-between gap-3">
              <span className="text-[0.67rem] font-semibold text-slate-400">
                Need to inspect the resource first?
              </span>
              <ResourceYAML
                item={item}
                size="xs"
                readOnly
                triggerComponent={
                  <Button
                    type="button"
                    variant="default"
                    size="compact-xs"
                    leftSection={<FileText size={11} strokeWidth={2.25} />}
                  >
                    Review YAML
                  </Button>
                }
              />
            </div>

            <div className="rounded-xl border border-slate-200 bg-slate-50/60 px-3.5 py-3">
              <Switch
                autoFocus
                checked={isConfirmed}
                disabled={mutationDelete.isPending}
                color="red.8"
                size="sm"
                label="I understand that this action is permanent"
                description={`Confirm deletion of ${item.kind} “${metadata.name}”`}
                onChange={(event) => {
                  setIsConfirmed(event.currentTarget.checked);
                  if (mutationDelete.isError) mutationDelete.reset();
                }}
                styles={{
                  label: {
                    color: "#334155",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                  },
                  description: {
                    color: "#94a3b8",
                    fontSize: "0.67rem",
                    fontWeight: 600,
                    marginTop: 2,
                  },
                }}
              />
            </div>

            {mutationDelete.isError && (
              <div className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-3">
                <p className="text-[0.72rem] font-bold text-red-700">
                  Deletion failed
                </p>
                <p className="mt-1 line-clamp-2 text-[0.68rem] font-semibold text-red-600/80">
                  {errorMessage}
                </p>
              </div>
            )}
          </div>

          <footer className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-4 py-3 sm:px-5">
            <Button
              type="button"
              variant="default"
              disabled={mutationDelete.isPending}
              onClick={handleClose}
            >
              Cancel
            </Button>
            <Button
              type="button"
              color="red.8"
              disabled={!isConfirmed || mutationDelete.isPending}
              loading={mutationDelete.isPending}
              leftSection={
                mutationDelete.isPending ? (
                  <Loader2 size={13} className="animate-spin" />
                ) : (
                  <Trash2 size={13} strokeWidth={2.25} />
                )
              }
              onClick={() => mutationDelete.mutate()}
            >
              {mutationDelete.isPending
                ? "Deleting…"
                : `Delete ${item.kind}`}
            </Button>
          </footer>
        </div>
      </Modal>
    </>
  );
};

export default DeleteResource;
