import { onError } from "@/utils";
import {
  getPB,
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
import { Button, CopyButton, Drawer, Tooltip } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  Check,
  Copy,
  FileText,
  Loader2,
  Moon,
  Save,
  Sun,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
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
}) => {
  const [viewMode, setViewMode] = React.useState<ViewMode>(0);
  const [colorScheme, setColorScheme] = React.useState<"dark" | "light">(
    "dark",
  );
  const { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const [cur, setCur] = React.useState<Resource>(item);
  const [isChanged, setIsChanged] = React.useState(false);

  React.useEffect(() => {
    try {
      // @ts-ignore
      const changed = !getPB(item)[`${item.kind}_Spec`].equals(
        item.spec,
        cur.spec,
      );
      setIsChanged(changed);
    } catch {
      setIsChanged(false);
    }
  }, [cur, item]);

  const mutationUpdate = useMutation({
    mutationFn: async (req: Resource) => {
      // @ts-ignore
      const { response } =
        // @ts-ignore
        await getResourceClient(cur)[`update${req.kind}`](cur);
      return response as Resource;
    },
    onSuccess: (response) => {
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
        ? resourceSpecToJSON(item)
        : resourceSpecToYAML(item),
    )
    .with(2, () =>
      props.mode === "json"
        ? resourceStatusToJSON(item)
        : resourceStatusToYAML(item),
    )
    .with(3, () => resourceMetadataToYAML(item))
    .otherwise(() =>
      props.mode === "json" ? resourceToJSON(item) : resourceToYAML(item),
    );

  const activeSchemaMode = VIEW_MODES.find(
    (m) => m.value === viewMode,
  )!.schemaMode;
  const isReadOnly = props.readOnly || item.metadata?.isSystem;

  return (
    <>
      {props.triggerComponent ? (
        <span onClick={open} className="cursor-pointer">
          {props.triggerComponent}
        </span>
      ) : (
        <Button
          size={`compact-xs`}
          variant={`outline`}
          onClick={open}
          className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
        >
          <FileText size={11} strokeWidth={2.5} />
          YAML
        </Button>
      )}

      <Drawer
        opened={opened}
        onClose={close}
        size="lg"
        withCloseButton={false}
        padding={0}
        styles={{
          body: {
            padding: 0,
            height: "100%",
            display: "flex",
            flexDirection: "column",
          },
          content: { display: "flex", flexDirection: "column" },
        }}
      >
        <div className="flex flex-col h-full bg-white">
          <div className="flex items-center justify-between px-5 py-3 shrink-0 border-b border-slate-200 bg-white">
            <div className="flex items-center gap-2 min-w-0">
              <FileText
                size={13}
                className="text-slate-400 shrink-0"
                strokeWidth={2}
              />
              <span className="text-[0.78rem] font-bold text-slate-700 truncate">
                {item.metadata?.name}
              </span>
              <span className="text-[0.68rem] font-semibold text-slate-400 shrink-0">
                {item.kind}
              </span>
              {item.metadata?.isSystem && (
                <Tooltip
                  label="System resource created by the cluster"
                  withArrow
                >
                  <span className="inline-flex items-center px-1.5 py-px text-[0.6rem] font-bold uppercase tracking-wider rounded border border-blue-200 text-blue-600 bg-blue-50 leading-none shrink-0">
                    System
                  </span>
                </Tooltip>
              )}
            </div>

            <div className="flex items-center gap-1 shrink-0">
              <Tooltip
                label={colorScheme === "dark" ? "Light editor" : "Dark editor"}
                withArrow
              >
                <button
                  onClick={() =>
                    setColorScheme((s) => (s === "dark" ? "light" : "dark"))
                  }
                  className="flex items-center justify-center w-7 h-7 rounded-md text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer"
                >
                  {colorScheme === "dark" ? (
                    <Sun size={13} strokeWidth={2} />
                  ) : (
                    <Moon size={13} strokeWidth={2} />
                  )}
                </button>
              </Tooltip>
              <button
                onClick={close}
                className="flex items-center justify-center w-7 h-7 rounded-md text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer"
              >
                <X size={13} strokeWidth={2} />
              </button>
            </div>
          </div>

          <div className="flex items-center justify-between px-4 py-2 shrink-0 border-b border-slate-100 bg-slate-50/60">
            <Button.Group>
              {VIEW_MODES.map((m) => (
                <Button
                  key={m.value}
                  onClick={() => setViewMode(m.value)}
                  styles={{
                    root: {
                      height: "26px",
                      fontSize: "0.7rem",
                      fontWeight: 700,
                      padding: "0 10px",
                      backgroundColor:
                        viewMode === m.value ? "#0f172a" : "#ffffff",
                      color: viewMode === m.value ? "#ffffff" : "#64748b",
                      border: "none",
                      borderRadius: 0,
                      transition: "background-color 150ms, color 150ms",
                      "&:hover": {
                        backgroundColor:
                          viewMode === m.value ? "#1e293b" : "#f8fafc",
                        color: viewMode === m.value ? "#ffffff" : "#0f172a",
                      },
                    },
                  }}
                >
                  {m.label}
                </Button>
              ))}
            </Button.Group>

            <CopyButton value={value}>
              {({ copied, copy }) => (
                <button
                  onClick={copy}
                  className={twMerge(
                    "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.7rem] font-bold border transition-colors duration-150 cursor-pointer",
                    copied
                      ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                      : "bg-white text-slate-500 border-slate-200 hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50",
                  )}
                >
                  <AnimatePresence initial={false} mode="popLayout">
                    <motion.span
                      key={copied ? "copied" : "copy"}
                      initial={{ y: 6, opacity: 0 }}
                      animate={{ y: 0, opacity: 1 }}
                      exit={{ y: -6, opacity: 0 }}
                      transition={{ duration: 0.12 }}
                      className="flex items-center gap-1.5"
                    >
                      {copied ? (
                        <>
                          <Check size={11} strokeWidth={2.5} /> Copied
                        </>
                      ) : (
                        <>
                          <Copy size={11} strokeWidth={2.5} /> Copy
                        </>
                      )}
                    </motion.span>
                  </AnimatePresence>
                </button>
              )}
            </CopyButton>
          </div>

          <AnimatePresence>
            {isChanged && !isReadOnly && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.15 }}
                className="overflow-hidden shrink-0"
              >
                <div className="flex items-center justify-between px-4 py-2 bg-amber-50 border-b border-amber-200">
                  <div className="flex items-center gap-1.5">
                    <span className="w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />
                    <span className="text-[0.72rem] font-semibold text-amber-700">
                      Unsaved changes
                    </span>
                  </div>
                  <button
                    onClick={() => mutationUpdate.mutate(cur)}
                    disabled={mutationUpdate.isPending}
                    className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.7rem] font-bold bg-amber-600 text-white hover:bg-amber-500 transition-colors duration-150 cursor-pointer disabled:opacity-60"
                  >
                    {mutationUpdate.isPending ? (
                      <>
                        <Loader2
                          size={11}
                          className="animate-spin"
                          strokeWidth={2.5}
                        />{" "}
                        Saving…
                      </>
                    ) : (
                      <>
                        <Save size={11} strokeWidth={2.5} /> Save
                      </>
                    )}
                  </button>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="flex-1 overflow-hidden p-3">
            <Editor
              item={props.item}
              mode={props.mode === "json" ? "json" : "yaml"}
              onResourceChange={(n) => setCur(n)}
              value={value}
              readOnly={isReadOnly}
              colorScheme={colorScheme}
              schemaMode={activeSchemaMode}
            />
          </div>

          {isReadOnly && (
            <div className="shrink-0 px-4 py-2.5 border-t border-slate-100 bg-slate-50/60 flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-slate-300 shrink-0" />
              <span className="text-[0.68rem] font-semibold text-slate-400">
                Read-only
                {item.metadata?.isSystem ? " · System resource" : ""}
              </span>
            </div>
          )}
        </div>
      </Drawer>
    </>
  );
};

export default ResourceYAML;
