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
import { AlertTriangle, Loader2, X } from "lucide-react";
import * as React from "react";
import { MdDelete } from "react-icons/md";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
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
}) => {
  const { item } = props;
  const [isDeletable, setIsDeletable] = React.useState(false);
  const navigate = useNavigate();
  const [opened, { open, close }] = useDisclosure(false);

  const handleClose = () => {
    setIsDeletable(false);
    close();
  };

  const mutationDelete = useMutation({
    mutationFn: async () => {
      // @ts-ignore
      await getResourceClient(item)[`delete${item.kind}`]({
        uid: item.metadata?.uid,
      });
    },
    onSuccess: () => {
      setIsDeletable(false);
      close();
      invalidateResourceList(item);
      props.onSuccess?.();
      if (!props.doNotNavigateAfter) {
        navigate(getResourceListPathFromResource(item));
      }
      toast.success(`${item.kind} ${item.metadata!.name} deleted`);
    },
    onError,
  });

  if (item.metadata?.isSystem) return null;

  return (
    <>
      <Button
        color={props.btnColor ?? "red.8"}
        onClick={open}
        size={props.btnSize}
        variant={props.btnVariant}
      >
        <MdDelete />
      </Button>

      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        withCloseButton={false}
        padding={0}
        styles={{
          content: {
            borderRadius: "12px",
            border: "1px solid #e2e8f0",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex flex-col">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 bg-slate-50/60">
            <div className="flex items-center gap-2">
              <AlertTriangle
                size={15}
                className="text-red-500 shrink-0"
                strokeWidth={2.5}
              />
              <span className="text-[0.82rem] font-bold text-slate-800">
                Delete {item.kind}
              </span>
            </div>
            <button
              onClick={handleClose}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors duration-150 cursor-pointer"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          </div>

          <div className="px-5 py-4 flex flex-col gap-4">
            <p className="text-[0.82rem] font-semibold text-slate-600">
              This action is permanent and cannot be undone. The following
              resource will be permanently deleted.
            </p>

            <div className="rounded-lg border border-slate-200 bg-slate-50 overflow-hidden">
              <div className="grid grid-cols-[80px_1fr] text-[0.75rem]">
                <span className="px-3 py-2 font-bold uppercase tracking-[0.05em] text-slate-400 border-b border-slate-200">
                  Name
                </span>
                <span className="px-3 py-2 font-semibold text-slate-700 font-mono border-b border-slate-200">
                  <CopyText value={item.metadata!.name} />
                </span>
                <span className="px-3 py-2 font-bold uppercase tracking-[0.05em] text-slate-400">
                  UID
                </span>
                <span className="px-3 py-2 font-semibold text-slate-500 font-mono truncate">
                  <CopyText value={item.metadata!.uid} />
                </span>
              </div>
            </div>

            <div className="flex items-center justify-between">
              <ResourceYAML item={item} size="xs" />
            </div>

            <Switch
              checked={isDeletable}
              onChange={(e) => setIsDeletable(e.currentTarget.checked)}
              color="red"
              size="md"
              label={
                <span className="text-[0.78rem] font-semibold text-slate-600">
                  I understand, permanently delete this {item.kind}
                </span>
              }
            />
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-slate-200 bg-slate-50/60">
            <button
              onClick={handleClose}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.78rem] font-bold text-slate-600 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer"
            >
              Cancel
            </button>

            <button
              disabled={!isDeletable || mutationDelete.isPending}
              onClick={() => mutationDelete.mutate()}
              className={twMerge(
                "flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.78rem] font-bold",
                "border transition-colors duration-150",
                "disabled:opacity-40 disabled:cursor-not-allowed",
                isDeletable
                  ? "bg-red-600 text-white border-red-600 hover:bg-red-700 hover:border-red-700 shadow-[0_2px_8px_rgba(220,38,38,0.3)]"
                  : "bg-red-100 text-red-400 border-red-200 cursor-not-allowed",
              )}
            >
              {mutationDelete.isPending ? (
                <>
                  <Loader2
                    size={12}
                    className="animate-spin"
                    strokeWidth={2.5}
                  />{" "}
                  Deleting…
                </>
              ) : (
                <>
                  <MdDelete size={13} /> Delete {item.kind}
                </>
              )}
            </button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default DeleteResource;
