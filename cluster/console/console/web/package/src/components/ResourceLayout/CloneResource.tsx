import { onError } from "@/utils";
import {
  cloneResource,
  getClient,
  getPBFromAPI,
  getResourcePath,
  invalidateResourceList,
  Resource,
} from "@/utils/pb";
import { Button, Modal, TextInput } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { Copy, Loader2, X } from "lucide-react";
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

const makeCloneName = (name: string) => {
  const dot = name.indexOf(".");
  if (dot === -1) return `${name}-clone`;
  return `${name.slice(0, dot)}-clone${name.slice(dot)}`;
};

const CloneResource = (props: {
  item: Resource;
  opened?: boolean;
  onClose?: () => void;
  hideTrigger?: boolean;
}) => {
  const { item } = props;
  const [internalOpened, { open, close: closeInternal }] = useDisclosure(false);
  const opened = props.opened ?? internalOpened;
  const close = props.onClose ?? closeInternal;
  const navigate = useNavigate();

  const originalName = item.metadata?.name ?? "";
  const [name, setName] = React.useState(makeCloneName(originalName));

  React.useEffect(() => {
    if (opened) setName(makeCloneName(originalName));
  }, [opened, originalName]);

  const api = item.apiVersion.split("/")[0];
  const kind = item.kind;

  const mutation = useMutation({
    mutationFn: async () => {
      // @ts-ignore
      const next: Resource = getPBFromAPI(api)[kind]["create"]({
        apiVersion: item.apiVersion,
        kind: item.kind,
        metadata: { name: name.trim() },
        spec: {},
        status: {},
      });

      next.spec = cloneResource(item).spec;
      if (kind.endsWith("Secret")) {
        // @ts-ignore
        next["data"] = cloneResource(item)["data"];
      }

      // @ts-ignore
      const { response } = await getClient(api)[`create${kind}`](next);
      return response as Resource;
    },
    onSuccess: (response) => {
      if (!response) return;
      invalidateResourceList(response);
      close();
      toast.success(`${kind} cloned successfully`);
      navigate(getResourcePath(response));
    },
    onError: (err: unknown) => {
      if (err instanceof Error) toast.error(err.message);
      // @ts-ignore
      onError(err);
    },
  });

  const canSubmit = name.trim().length > 0 && name.trim() !== originalName;

  return (
    <>
      {!props.hideTrigger && (
        <Button
          size="compact-xs"
          variant="outline"
          leftSection={<Copy size={11} />}
          onClick={(e: React.MouseEvent) => {
            e.stopPropagation();
            open();
          }}
        >
          Clone
        </Button>
      )}

      <Modal
        opened={opened}
        onClose={close}
        size="md"
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
              <Copy
                size={14}
                className="text-slate-500 shrink-0"
                strokeWidth={2}
              />
              <span className="text-[0.82rem] font-bold text-slate-800">
                Clone {kind}
              </span>
              <span className="text-[0.7rem] font-semibold text-slate-400 font-mono">
                {originalName}
              </span>
            </div>
            <button
              onClick={close}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors duration-150 cursor-pointer"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          </div>

          <div className="px-5 py-5">
            <TextInput
              label="New name"
              description="Set a unique name for the cloned resource"
              placeholder={makeCloneName(originalName)}
              value={name}
              onChange={(e) => setName(e.currentTarget.value)}
              error={
                name.trim() === originalName
                  ? "Name must differ from the original"
                  : undefined
              }
              data-autofocus
            />
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-slate-200 bg-slate-50/60">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              onClick={close}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="filled"
              color="dark"
              size="sm"
              disabled={!canSubmit || mutation.isPending}
              loading={mutation.isPending}
              leftSection={
                mutation.isPending ? (
                  <Loader2
                    size={13}
                    strokeWidth={2.5}
                    className="animate-spin"
                  />
                ) : (
                  <Copy size={13} strokeWidth={2.5} />
                )
              }
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending ? "Cloning..." : "Clone"}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default CloneResource;
