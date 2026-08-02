import { onError } from "@/utils";
import {
  cloneResource,
  getResourceClient,
  invalidateResource,
  invalidateResourceList,
  Resource,
  resourceMetadataToYAML,
  resourceSpecToJSON,
  resourceSpecToYAML,
  resourceStatusToJSON,
  resourceStatusToYAML,
  resourceToJSON,
  resourceToYAML,
} from "@/utils/pb";
import {
  Button,
  CopyButton,
  Drawer,
  SegmentedControl,
  Tooltip,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { Check, Copy, FileText, Loader2, Save } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { match } from "ts-pattern";
import Editor from "../Editor";

type ViewMode = 0 | 1 | 2 | 3;

const VIEW_MODES: {
  value: ViewMode;
  label: string;
  schemaMode: "full" | "spec" | "status" | "metadata";
}[] = [
  { value: 0, label: "All", schemaMode: "full" },
  { value: 1, label: "Spec", schemaMode: "spec" },
  { value: 2, label: "Status", schemaMode: "status" },
  { value: 3, label: "Metadata", schemaMode: "metadata" },
];

const ResourceYAML = (props: {
  item: Resource;
  size?: "xs" | "small";
  btnItem?: boolean;
  triggerComponent?: React.ReactNode;
  onResourceChange?: (arg: Resource) => void;
  readOnly?: boolean;
  mode?: "json" | "yaml" | undefined;
  opened?: boolean;
  onClose?: () => void;
  hideTrigger?: boolean;
}) => {
  const [viewMode, setViewMode] = React.useState<ViewMode>(0);
  const { item } = props;
  const [internalOpened, { open, close: closeInternal }] = useDisclosure(false);
  const opened = props.opened ?? internalOpened;
  const close = props.onClose ?? closeInternal;
  const [cur, setCur] = React.useState<Resource>(() => cloneResource(item));
  const [baseline, setBaseline] = React.useState<Resource>(() =>
    cloneResource(item),
  );

  React.useEffect(() => {
    if (!opened) {
      const next = cloneResource(item);
      setCur(next);
      setBaseline(cloneResource(next));
    }
  }, [item, opened]);

  const isChanged = React.useMemo(
    () => resourceToJSON(cur) !== resourceToJSON(baseline),
    [baseline, cur],
  );

  const mutationUpdate = useMutation({
    mutationFn: async (req: Resource) => {
      // @ts-ignore
      const { response } =
        // @ts-ignore
        await getResourceClient(req)[`update${req.kind}`](req);
      return response as Resource;
    },
    onSuccess: (response) => {
      const next = cloneResource(response);
      setCur(next);
      setBaseline(cloneResource(next));
      props.onResourceChange?.(response);
      invalidateResource(response);
      invalidateResourceList(response);
      toast.success(
        `${response.kind} ${response.metadata?.name} updated successfully`,
      );
    },
    onError,
  });

  const value = match(viewMode)
    .with(1, () =>
      props.mode === "json"
        ? resourceSpecToJSON(cur)
        : resourceSpecToYAML(cur),
    )
    .with(2, () =>
      props.mode === "json"
        ? resourceStatusToJSON(cur)
        : resourceStatusToYAML(cur),
    )
    .with(3, () => resourceMetadataToYAML(cur))
    .otherwise(() =>
      props.mode === "json" ? resourceToJSON(cur) : resourceToYAML(cur),
    );

  const activeSchemaMode = VIEW_MODES.find(
    (m) => m.value === viewMode,
  )!.schemaMode;
  const isReadOnly = props.readOnly || item.metadata?.isSystem;

  return (
    <>
      {!props.hideTrigger && props.triggerComponent ? (
        <span onClick={open} className="cursor-pointer">
          {props.triggerComponent}
        </span>
      ) : !props.hideTrigger ? (
        <Button variant={`outline`} size={`compact-xs`} onClick={open}>
          <FileText size={11} strokeWidth={2.5} />
          YAML
        </Button>
      ) : null}

      <Drawer
        opened={opened}
        onClose={close}
        position="right"
        size="min(900px, 100vw)"
        padding="md"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <FileText
              size={15}
              className="shrink-0 text-slate-400"
              strokeWidth={2.25}
            />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              YAML
            </span>
            <span className="text-sm font-semibold text-slate-800 truncate">
              {item.metadata?.name}
            </span>
            <span className="hidden text-xs font-semibold text-slate-400 sm:inline">
              {item.kind}
            </span>
            {item.metadata?.isSystem && (
              <Tooltip label="System resource" withArrow>
                <span className="hidden items-center rounded-md border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-[0.6rem] font-bold uppercase tracking-wider text-blue-600 sm:inline-flex">
                  System
                </span>
              </Tooltip>
            )}
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: "16px",
            display: "flex",
            flexDirection: "column",
            backgroundColor: "#f8fafc",
          },
          content: {
            display: "flex",
            flexDirection: "column",
            borderLeft: "1px solid #e2e8f0",
          },
        }}
      >
        <div className="flex min-h-0 flex-1 flex-col gap-3">
          <div className="flex shrink-0 flex-col gap-2 rounded-xl border border-slate-200 bg-white p-2 shadow-sm sm:flex-row sm:items-center sm:justify-between">
            <SegmentedControl
              value={String(viewMode)}
              onChange={(v) => setViewMode(Number(v) as ViewMode)}
              data={VIEW_MODES.map((m) => ({
                label: m.label,
                value: String(m.value),
              }))}
              fullWidth
            />

            <CopyButton value={value}>
              {({ copied, copy }) => (
                <Button
                  onClick={copy}
                  variant={copied ? "light" : "default"}
                  color={copied ? "teal" : undefined}
                  size="compact-sm"
                  leftSection={
                    <AnimatePresence initial={false} mode="popLayout">
                      <motion.span
                        key={copied ? "check" : "copy"}
                        initial={{ y: 6, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        exit={{ y: -6, opacity: 0 }}
                        transition={{ duration: 0.12 }}
                        className="flex items-center"
                      >
                        {copied ? (
                          <Check size={12} strokeWidth={2.5} />
                        ) : (
                          <Copy size={12} strokeWidth={2.5} />
                        )}
                      </motion.span>
                    </AnimatePresence>
                  }
                  styles={{
                    root: {
                      fontFamily: "Ubuntu, sans-serif",
                      fontSize: "0.72rem",
                      fontWeight: 700,
                    },
                  }}
                >
                  {copied ? "Copied" : "Copy"}
                </Button>
              )}
            </CopyButton>
          </div>

          <div className="min-h-0 flex-1 overflow-hidden rounded-xl border border-slate-200 bg-white p-2 shadow-sm">
            <Editor
              item={cur}
              mode={props.mode === "json" ? "json" : "yaml"}
              onResourceChange={(n) => setCur(n)}
              value={value}
              readOnly={isReadOnly}
              schemaMode={activeSchemaMode}
              minHeight="calc(100dvh - 210px)"
              maxHeight="calc(100dvh - 210px)"
            />
          </div>

          <div className="flex min-h-9 shrink-0 items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2 shadow-sm">
            <div className="min-w-0">
              <AnimatePresence mode="wait" initial={false}>
                <motion.div
                  key={isReadOnly ? "readonly" : isChanged ? "changed" : "saved"}
                  initial={{ opacity: 0, y: 3 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -3 }}
                  transition={{ duration: 0.15 }}
                  className="flex items-center gap-2"
                >
                  <span
                    className={`h-1.5 w-1.5 shrink-0 rounded-full ${
                      isReadOnly
                        ? "bg-slate-300"
                        : isChanged
                          ? "bg-amber-500"
                          : "bg-emerald-500"
                    }`}
                  />
                  <span
                    className={`truncate text-[0.72rem] font-semibold ${
                      isChanged && !isReadOnly
                        ? "text-amber-700"
                        : "text-slate-500"
                    }`}
                  >
                    {isReadOnly
                      ? `Read-only${item.metadata?.isSystem ? " · System resource" : ""}`
                      : isChanged
                        ? "Unsaved changes"
                        : "No unsaved changes"}
                  </span>
                </motion.div>
              </AnimatePresence>
            </div>

            {!isReadOnly && (
              <Button
                size="compact-sm"
                variant="filled"
                color="dark"
                leftSection={
                  mutationUpdate.isPending ? (
                    <Loader2
                      size={11}
                      className="animate-spin"
                      strokeWidth={2.5}
                    />
                  ) : (
                    <Save size={11} strokeWidth={2.5} />
                  )
                }
                disabled={!isChanged || mutationUpdate.isPending}
                loading={mutationUpdate.isPending}
                onClick={() => mutationUpdate.mutate(cur)}
                styles={{
                  root: {
                    fontFamily: "Ubuntu, sans-serif",
                    fontSize: "0.7rem",
                    fontWeight: 700,
                  },
                }}
              >
                {mutationUpdate.isPending ? "Saving…" : "Save changes"}
              </Button>
            )}
          </div>
        </div>
      </Drawer>
    </>
  );
};

export default ResourceYAML;
