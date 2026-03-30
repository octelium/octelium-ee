import * as React from "react";

import { onError } from "@/utils";

import { useNavigate } from "react-router-dom";

import PageWrap from "@/components/PageWrap";
import {
  cloneResource,
  getAPI,
  getResourceClient,
  getResourcePath,
  invalidateResource,
  invalidateResourceList,
  Resource,
  resourceFromYAML,
  resourceToYAML,
} from "@/utils/pb";
import { useMutation } from "@tanstack/react-query";
import { useContextResource } from "./utils";

import { Button, Tabs } from "@mantine/core";
import { toast } from "sonner";
import ContainerGen from "../ContainerGen";
import MetadataEdit from "../MetadataEdit";
import ResourceEditor from "../ResourceEditor";

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
}) => {
  let [req, setReq] = React.useState<Resource>(props.item);
  let [curYAML, setCurYAML] = React.useState(resourceToYAML(req));
  let [isYAML, setIsYAML] = React.useState(false);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(cloneResource(data));
    }
  }, [data]);

  const navigate = useNavigate();

  const api = getAPI(req);

  if (!api) {
    return <></>;
  }

  const mutationUpdate = useMutation({
    mutationFn: async (req: Resource) => {
      let rsc: Resource | undefined;
      if (isYAML) {
        rsc = resourceFromYAML(curYAML);
      } else {
        rsc = req;
      }

      if (!rsc) {
        return req;
      }
      // @ts-ignore
      const { response } =
        // @ts-ignore
        await getResourceClient(rsc)[`update${req.kind}`](rsc);
      return response as Resource;
    },

    onSuccess: (response) => {
      if (props.onUpdateDone) {
        props.onUpdateDone(response);
      }
      invalidateResource(response);
      invalidateResourceList(response);
      if (!props.noPostUpdateNavigation) {
        navigate(getResourcePath(response));
      }

      if (!props.noPostUpdateToast) {
        toast.success(
          `${response.kind}: ${response.metadata?.name} successfully updated`,
        );
      }
    },
    onError,
  });

  if (!req) {
    return <></>;
  }

  /*
  if (req.metadata?.isSystem) {
    return (
      <div className="w-full flex items-center justify-center my-32 text-slate-700 text-2xl">
        This is a System Resource and Cannot be Edited
      </div>
    );
  }
  */

  return (
    <div className="w-full">
      <Tabs
        defaultValue="main"
        onChange={(v) => {
          setIsYAML(v === "yaml");
        }}
      >
        <Tabs.List className="mb-6">
          <Tabs.Tab
            value="main"
            onClick={() => {
              navigate(".");
            }}
          >
            Configuration
          </Tabs.Tab>

          <Tabs.Tab value="yaml">YAML</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="main">
          <div>
            <div>
              {!props.noMetadata && (
                <ContainerGen title="Metadata">
                  <MetadataEdit
                    isUpdateMode
                    item={data!}
                    onUpdate={(md) => {
                      let req2 = cloneResource(req);
                      req2.metadata = md;
                      setReq(cloneResource(req2));
                    }}
                  />
                </ContainerGen>
              )}
            </div>
            <div className="w-full my-12">
              {props.specComponent({
                item: data,
                onUpdate: (item) => {
                  let req2 = cloneResource(req);
                  req2.spec = item.spec;
                  if (item.kind.endsWith("Secret")) {
                    // @ts-ignore
                    req2["data"] = item["data"];
                  }
                  setReq(req2);
                },
              })}
            </div>
          </div>
        </Tabs.Panel>
        <Tabs.Panel value="yaml">
          <ResourceEditor
            item={req}
            onResourceChange={(item) => {
              setReq(cloneResource(item));
            }}
            onChange={(v) => {
              setCurYAML(v);
            }}
          />
        </Tabs.Panel>
      </Tabs>

      <div className="mt-24">
        <div className="flex flex-row justify-end items-center">
          <Button
            variant="outline"
            size={"lg"}
            onClick={() => {
              navigate(-1);
            }}
          >
            Cancel
          </Button>

          <Button
            size={"lg"}
            className="ml-4 shadow-lg"
            onClick={() => {
              if (req) {
                mutationUpdate.mutate(req);
              }
            }}
            loading={mutationUpdate.isPending}
          >
            Update
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
}) => {
  const ctx = useContextResource();
  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && (
        <ResourceEdit
          item={ctx.data}
          specComponent={props.specComponent}
          dataComponent={props.dataComponent}
        />
      )}
    </PageWrap>
  );
};

export default ResourceEditPage;
