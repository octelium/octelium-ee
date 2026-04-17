import { onError } from "@/utils";
import {
  APIKind,
  cloneResource,
  getAPIKindFromPath,
  getClient,
  getPBFromAPI,
  getResourcePath,
  invalidateResourceList,
  Resource,
  resourceFromYAML,
  resourceToYAML,
} from "@/utils/pb";
import { Tabs } from "@mantine/core";
import { useMutation, useQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { FileCode, Loader2, Plus, Settings, X } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import ContainerGen from "../ContainerGen";
import MetadataEdit from "../MetadataEdit";
import ResourceEditor from "../ResourceEditor";

const createResource = (apiKind: APIKind): Resource => {
  // @ts-ignore
  return getPBFromAPI(apiKind.api)[`${apiKind.kind}`]["create"]({
    apiVersion: `${apiKind.api}/v1`,
    kind: apiKind.kind,
    metadata: {},
    spec: {},
    status: {},
  });
};

const ResourceCreatePage = (props: {
  specComponent: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  dataComponent?: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  createResource?: () => Resource;
}) => {
  const loc = useLocation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const apiKind = getAPIKindFromPath(loc.pathname)!;
  const cloneUID = searchParams.get("cloneRef.uid") ?? undefined;

  const [req, setReq] = React.useState<Resource>(
    props.createResource ? props.createResource() : createResource(apiKind),
  );
  const [curYAML, setCurYAML] = React.useState(() => resourceToYAML(req));
  const [activeTab, setActiveTab] = React.useState<"main" | "yaml">("main");
  const [yamlParseError, setYamlParseError] = React.useState<string | null>(
    null,
  );

  useQuery({
    queryKey: ["resourceClone", apiKind.api, apiKind.kind, cloneUID],
    enabled: !!cloneUID,
    queryFn: async () => {
      // @ts-ignore
      const { response } = await getClient(apiKind.api)[`get${apiKind.kind}`]({
        uid: cloneUID,
      } as any);
      return response as Resource;
    },
    select: (source) => {
      if (!source) return;
      const next = cloneResource(req);
      next.spec = source.spec;
      setReq(next);
      setCurYAML(resourceToYAML(next));
    },
  });

  const mutation = useMutation({
    mutationFn: async () => {
      let rsc: Resource | undefined;

      if (activeTab === "yaml") {
        rsc = resourceFromYAML(curYAML);
        if (!rsc) {
          throw new Error("Invalid YAML — could not parse resource.");
        }
      } else {
        rsc = req;
      }

      // @ts-ignore
      const { response } = await getClient(apiKind.api)[
        // @ts-ignore
        `create${apiKind.kind}`
      ](rsc);
      return response as Resource;
    },
    onSuccess: (response) => {
      if (!response) return;
      invalidateResourceList(response);
      navigate(getResourcePath(response));
    },
    onError: (err: unknown) => {
      if (err instanceof Error) {
        toast.error(err.message);
      }
      // @ts-ignore
      onError(err);
    },
  });

  const handleYAMLChange = (v: string) => {
    setCurYAML(v);
    const parsed = resourceFromYAML(v);
    setYamlParseError(parsed ? null : "Invalid YAML — cannot parse resource");
  };

  const canSubmit = activeTab === "yaml" ? yamlParseError === null : true;

  return (
    <div className="w-full flex flex-col gap-6">
      <Tabs
        value={activeTab}
        onChange={(v) => setActiveTab(v as "main" | "yaml")}
      >
        <Tabs.List mb="lg">
          <Tabs.Tab
            value="main"
            leftSection={<Settings size={13} strokeWidth={2.5} />}
          >
            Configuration
          </Tabs.Tab>
          <Tabs.Tab
            value="yaml"
            leftSection={<FileCode size={13} strokeWidth={2.5} />}
          >
            YAML
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="main">
          <div className="flex flex-col gap-8">
            {req.metadata && (
              <ContainerGen title="Metadata">
                <MetadataEdit
                  item={req}
                  onUpdate={(v) => {
                    const next = cloneResource(req);
                    next.metadata = v;
                    setReq(next);
                  }}
                />
              </ContainerGen>
            )}

            {props.specComponent && (
              <ContainerGen title="Spec">
                {props.specComponent({
                  item: req,
                  onUpdate: (item) => {
                    const next = cloneResource(req);
                    next.spec = item.spec;
                    if (item.kind.endsWith("Secret")) {
                      // @ts-ignore
                      next["data"] = item["data"];
                    }
                    setReq(next);
                  },
                })}
              </ContainerGen>
            )}
          </div>
        </Tabs.Panel>

        <Tabs.Panel value="yaml">
          <div className="flex flex-col gap-3">
            <ResourceEditor
              item={req}
              onResourceChange={(item) => {
                setReq(cloneResource(item));
              }}
              onChange={handleYAMLChange}
            />

            <AnimatePresence>
              {yamlParseError && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.15 }}
                  className="overflow-hidden"
                >
                  <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-50 border border-red-200">
                    <X
                      size={12}
                      className="text-red-500 shrink-0"
                      strokeWidth={2.5}
                    />
                    <span className="text-[0.72rem] font-semibold text-red-700">
                      {yamlParseError}
                    </span>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </Tabs.Panel>
      </Tabs>

      <div className="flex items-center justify-between pt-4 border-t border-slate-200">
        {mutation.isError && (
          <span className="text-[0.72rem] font-semibold text-red-600">
            Creation failed — check the form and try again.
          </span>
        )}
        <div className="flex-1" />

        <div className="flex items-center gap-2">
          <button
            onClick={() => navigate(-1)}
            disabled={mutation.isPending}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-[0.78rem] font-bold text-slate-600 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer disabled:opacity-50"
          >
            <X size={13} strokeWidth={2.5} />
            Cancel
          </button>

          <button
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending || !canSubmit}
            className={twMerge(
              "flex items-center gap-1.5 px-4 py-2 rounded-lg text-[0.78rem] font-bold",
              "bg-slate-900 text-white border border-slate-900",
              "hover:bg-slate-800 transition-colors duration-150",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              "shadow-[0_2px_8px_rgba(15,23,42,0.18)]",
            )}
          >
            {mutation.isPending ? (
              <>
                <Loader2 size={13} className="animate-spin" strokeWidth={2.5} />
                Creating…
              </>
            ) : (
              <>
                <Plus size={13} strokeWidth={2.5} />
                Create {apiKind.kind}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ResourceCreatePage;
