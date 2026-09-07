import { onError } from "@/utils";
import {
  APIKind,
  cloneResource,
  createResourcePB,
  getAPIKindFromPath,
  getResourcePath,
  getResourcePB,
  invalidateResourceList,
  newResourcePB,
  Resource,
  resourceFromYAML,
} from "@/utils/pb";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import ResourceForm from "./ResourceForm";

type SpecComponent = React.ComponentType<{
  item: Resource;
  onUpdate: (item: Resource) => void;
}>;

const blankResource = (apiKind: APIKind): Resource =>
  newResourcePB(apiKind.api, apiKind.kind, {
    apiVersion: `${apiKind.api}/v1`,
    kind: apiKind.kind,
    metadata: {},
    spec: {},
    status: {},
  });

const ResourceCreatePage = (props: {
  specComponent: SpecComponent;
  dataComponent?: SpecComponent;
  createResource?: () => Resource;
  onCreated?: (item: Resource) => void;
  onCancel?: () => void;
}) => {
  const loc = useLocation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const apiKind = getAPIKindFromPath(loc.pathname)!;
  const cloneUID = searchParams.get("cloneRef.uid") ?? undefined;

  const [seed, setSeed] = React.useState<Resource>(() =>
    props.createResource ? props.createResource() : blankResource(apiKind),
  );

  const cloneQuery = useQuery({
    queryKey: ["resourceClone", apiKind.api, apiKind.kind, cloneUID],
    enabled: !!cloneUID,
    queryFn: async () => {
      const { response } = await getResourcePB(apiKind.api, apiKind.kind, {
        uid: cloneUID,
      });
      return response as Resource;
    },
  });

  const clonedUID = React.useRef<string | undefined>(undefined);

  React.useEffect(() => {
    const source = cloneQuery.data;
    if (!source || clonedUID.current === cloneUID) return;
    clonedUID.current = cloneUID;
    setSeed((current) => {
      const next = cloneResource(current);
      next.spec = source.spec;
      return next;
    });
  }, [cloneQuery.data, cloneUID]);

  const mutation = useMutation({
    mutationFn: async (arg: {
      req: Resource;
      yaml: string;
      isYAML: boolean;
    }) => {
      const resource = arg.isYAML ? resourceFromYAML(arg.yaml) : arg.req;
      if (!resource)
        throw new Error("Invalid YAML — could not parse resource.");
      const { response } = await createResourcePB(
        apiKind.api,
        apiKind.kind,
        resource,
      );
      return response as Resource;
    },
    onSuccess: (response) => {
      if (!response) return;
      invalidateResourceList(response);
      if (props.onCreated) {
        props.onCreated(response);
        return;
      }
      navigate(getResourcePath(response));
    },
    onError: (err: unknown) => onError(err as any),
  });

  return (
    <ResourceForm
      item={seed}
      specComponent={props.specComponent}
      dataComponent={props.dataComponent}
      submitIcon={<Plus size={13} strokeWidth={2.25} />}
      submitLabel={`Create ${apiKind.kind}`}
      submitPendingLabel="Creating…"
      isPending={mutation.isPending}
      isError={mutation.isError}
      errorLabel="Creation failed — check the form and try again."
      onCancel={() => {
        if (props.onCancel) {
          props.onCancel();
          return;
        }
        navigate("..", {
          relative: "path",
          state: loc.state,
          preventScrollReset: true,
        });
      }}
      onSubmit={(req, yaml, isYAML) => mutation.mutate({ req, yaml, isYAML })}
    />
  );
};

export default ResourceCreatePage;
