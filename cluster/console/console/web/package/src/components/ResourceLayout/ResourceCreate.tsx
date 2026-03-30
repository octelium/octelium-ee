import * as React from "react";

import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

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
import { useMutation } from "@tanstack/react-query";

import { Button, Tabs } from "@mantine/core";
import ContainerGen from "../ContainerGen";
import MetadataEdit from "../MetadataEdit";
import ResourceEditor from "../ResourceEditor";

const createResource = (apiKind: APIKind): Resource => {
  const req = {
    apiVersion: `${apiKind?.api}/v1`,
    kind: apiKind?.kind,
    metadata: {},
    spec: {},
    status: {},
  };

  // @ts-ignore
  return getPBFromAPI(apiKind.api)[`${apiKind.kind}`]["create"](req);
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
  const [searchParams, _] = useSearchParams();

  const apiKind = getAPIKindFromPath(loc.pathname)!;

  let [req, setReq] = React.useState<Resource>(
    props.createResource ? props.createResource() : createResource(apiKind),
  );

  const cloneUID = searchParams.get("cloneRef.uid") ?? undefined;

  /*
  const qryFrom = useQuery({
    queryKey: [getGetKeyFromPath(loc.pathname), cloneUID],
    enabled: !!cloneUID,
    queryFn: () => {
      //@ts-ignore
      const { response } = getClient(apiKind.api)[`get${apiKind.kind}`]({
        uid: cloneUID,
      } as any);

      return response as Resource;
    },

    select: (d) => {
      setReq(d);
    },
  });
  */

  let [curYAML, setCurYAML] = React.useState(resourceToYAML(req));
  let [isYAML, setIsYAML] = React.useState(false);

  const navigate = useNavigate();

  const mutation = useMutation({
    mutationFn: async () => {
      let rsc: Resource | undefined;
      if (isYAML) {
        rsc = resourceFromYAML(curYAML);
      } else {
        rsc = req;
      }

      if (!rsc) {
        return { resource: req };
      }

      // @ts-ignore
      const { response } = await getClient(apiKind.api)[
        // @ts-ignore
        `create${apiKind.kind}`
      ](rsc);

      return { response: response as Resource };
    },
    onSuccess: ({ response }) => {
      if (!response) {
        return;
      }

      invalidateResourceList(response);
      navigate(getResourcePath(response));
    },
    onError,
  });

  const mutationFrom = useMutation({
    mutationFn: async () => {
      if (!cloneUID) {
        return;
      }
      //@ts-ignore
      const { response } = getClient(apiKind.api)[`get${apiKind.kind}`]({
        uid: cloneUID,
      } as any);

      return response as Resource;
    },
    onSuccess: (r) => {
      if (!r) {
        return;
      }
      let c = cloneResource(req);
      c.spec = r.spec;

      setReq(c);
    },
  });

  React.useEffect(() => {
    mutationFrom.mutate();
  }, []);

  return (
    <div>
      <Tabs
        defaultValue="main"
        onChange={(v) => {
          setIsYAML(v === "yaml");
        }}
      >
        <Tabs.List className="mb-2">
          <Tabs.Tab
            value="main"
            onClick={() => {
              navigate(".");
            }}
          >
            Main
          </Tabs.Tab>

          <Tabs.Tab value="yaml">YAML</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="main">
          <div>
            <div className="mb-12">
              {req.metadata && (
                <ContainerGen title="Metadata">
                  <MetadataEdit
                    item={req}
                    onUpdate={(v) => {
                      let req2 = cloneResource(req);
                      req2.metadata = v;
                      setReq(req2);
                    }}
                  />
                </ContainerGen>
              )}
            </div>

            <div className="w-full">
              {props.specComponent && (
                <ContainerGen title="Spec">
                  {props.specComponent({
                    item: req,
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
                </ContainerGen>
              )}
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

      <div>
        <div className="flex flex-row items-center justify-end mt-8">
          <Button
            variant={"outline"}
            size={"lg"}
            onClick={() => {
              navigate(-1);
            }}
          >
            Cancel
          </Button>
          <Button
            className="ml-4 shadow-lg"
            size={"lg"}
            onClick={() => {
              mutation.mutate();
            }}
            loading={mutation.isPending}
          >
            Create
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ResourceCreatePage;
