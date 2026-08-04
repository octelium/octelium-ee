import PageWrap from "@/components/PageWrap";
import { onError } from "@/utils";
import {
  cloneResource,
  getResourceClient,
  getResourcePath,
  invalidateResource,
  invalidateResourceList,
  Resource,
  resourceFromYAML,
  resourceToYAML,
} from "@/utils/pb";
import { Button, SegmentedControl } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { FileCode, Loader2, Save, Settings, X } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import ContainerGen from "../ContainerGen";
import MetadataEdit from "../MetadataEdit";
import ResourceEditor from "../ResourceEditor";
import { useContextResource } from "./utils";

export const ResourceEdit = (props: {
  item: Resource;
  specComponent: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  dataComponent?: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  noMetadata?: boolean;
  onUpdateDone?: (item: Resource) => void;
  noPostUpdateNavigation?: boolean;
  noPostUpdateToast?: boolean;
  readOnly?: boolean;
}) => {
  const [req, setReq] = React.useState<Resource>(cloneResource(props.item));
  const [curYAML, setCurYAML] = React.useState(() =>
    resourceToYAML(props.item),
  );
  const [activeTab, setActiveTab] = React.useState<"main" | "yaml">("main");
  const [yamlParseError, setYamlParseError] = React.useState<string | null>(
    null,
  );

  const data = props.item;
  const navigate = useNavigate();
  const location = useLocation();

  React.useEffect(() => {
    if (data) {
      const cloned = cloneResource(data);
      setReq(cloned);
      setCurYAML(resourceToYAML(cloned));
      setYamlParseError(null);
    }
  }, [data]);

  const mutationUpdate = useMutation({
    mutationFn: async (req: Resource) => {
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
      const { response } =
        // @ts-ignore
        await getResourceClient(rsc)[`update${rsc.kind}`](rsc);
      return response as Resource;
    },
    onSuccess: (response) => {
      props.onUpdateDone?.(response);
      invalidateResource(response);
      invalidateResourceList(response);
      if (!props.noPostUpdateNavigation) {
        navigate(getResourcePath(response), {
          state: location.state,
          preventScrollReset: true,
        });
      }
      if (!props.noPostUpdateToast) {
        toast.success(
          `${response.kind} ${response.metadata?.name} updated successfully`,
        );
      }
    },
    onError: (err: unknown) => onError(err as any),
  });

  const handleYAMLChange = (v: string) => {
    setCurYAML(v);
    const parsed = resourceFromYAML(v);
    setYamlParseError(parsed ? null : "Invalid YAML — cannot parse resource");
    if (parsed) setReq(cloneResource(parsed));
  };

  const handleTabChange = (value: string) => {
    const nextTab = value as "main" | "yaml";
    if (nextTab === "yaml") {
      setCurYAML(resourceToYAML(req));
      setYamlParseError(null);
    } else {
      const parsed = resourceFromYAML(curYAML);
      if (!parsed) {
        setYamlParseError("Invalid YAML — cannot parse resource");
        return;
      }
      setReq(cloneResource(parsed));
    }
    setActiveTab(nextTab);
  };

  const handleCancel = () => {
    if (location.pathname.endsWith("/edit")) {
      navigate("..", {
        relative: "path",
        state: location.state,
        preventScrollReset: true,
      });
      return;
    }
    navigate(-1);
  };

  const canSubmit = activeTab === "yaml" ? yamlParseError === null : true;
  const isDirty = resourceToYAML(req) !== resourceToYAML(data);

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="flex items-center">
        <SegmentedControl
          value={activeTab}
          onChange={handleTabChange}
          disabled={props.readOnly}
          data={[
            {
              value: "main",
              label: (
                <span className="flex items-center gap-1.5 px-1">
                  <Settings size={13} strokeWidth={2.5} />
                  Configuration
                </span>
              ),
            },
            {
              value: "yaml",
              label: (
                <span className="flex items-center gap-1.5 px-1">
                  <FileCode size={13} strokeWidth={2.5} />
                  YAML
                </span>
              ),
            },
          ]}
        />
      </div>

      {activeTab === "main" ? (
        <div className="flex flex-col gap-8">
          {!props.noMetadata && (
            <ContainerGen title="Metadata">
              <fieldset disabled={props.readOnly} className={props.readOnly ? "opacity-55" : undefined}>
                <MetadataEdit
                  isUpdateMode
                  item={req}
                  onUpdate={(md) => {
                    const next = cloneResource(req);
                    next.metadata = md;
                    setReq(next);
                  }}
                />
              </fieldset>
            </ContainerGen>
          )}

          {props.specComponent && (
            <ContainerGen title="Spec">
              {React.createElement(props.specComponent, {
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

          {props.dataComponent && (
            <ContainerGen title="Data">
              {React.createElement(props.dataComponent, {
                item: req,
                onUpdate: (item) => {
                  const next = cloneResource(req);
                  const nextWithData = next as Resource & { data?: unknown };
                  const itemWithData = item as Resource & { data?: unknown };
                  nextWithData.data = itemWithData.data;
                  setReq(next);
                },
              })}
            </ContainerGen>
          )}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <ResourceEditor
            item={req}
            value={curYAML}
            onResourceChange={(item) => setReq(cloneResource(item))}
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
      )}

      <div className="flex items-center justify-between pt-4 border-t border-slate-200 mt-2">
        {mutationUpdate.isError && (
          <span className="text-[0.72rem] font-semibold text-red-600">
            Update failed — check the form and try again.
          </span>
        )}
        <div className="flex-1" />

        <div className="flex items-center gap-2">
          <Button
            variant="default"
            leftSection={<X size={13} strokeWidth={2.5} />}
            disabled={mutationUpdate.isPending}
            onClick={handleCancel}
          >
            Cancel
          </Button>

          <Button
            variant="filled"
            color="dark"
            leftSection={
              mutationUpdate.isPending ? (
                <Loader2 size={13} className="animate-spin" strokeWidth={2.5} />
              ) : (
                <Save size={13} strokeWidth={2.5} />
              )
            }
            disabled={
              props.readOnly || mutationUpdate.isPending || !canSubmit || !isDirty
            }
            loading={mutationUpdate.isPending}
            onClick={() => mutationUpdate.mutate(req)}
          >
            {mutationUpdate.isPending ? "Saving…" : "Save changes"}
          </Button>
        </div>
      </div>
    </div>
  );
};

const ResourceEditPage = (props: {
  specComponent: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  dataComponent?: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  readOnly?: boolean;
}) => {
  const ctx = useContextResource();
  if (!ctx) return null;

  return (
    <PageWrap qry={ctx}>
      {ctx.data && (
        <ResourceEdit
          item={ctx.data}
          specComponent={props.specComponent}
          dataComponent={props.dataComponent}
          readOnly={props.readOnly}
        />
      )}
    </PageWrap>
  );
};

export default ResourceEditPage;
