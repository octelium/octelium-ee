import PageWrap from "@/components/PageWrap";
import { onError } from "@/utils";
import {
  getResourcePath,
  invalidateResource,
  invalidateResourceList,
  Resource,
  resourceFromYAML,
  updateResourcePB,
} from "@/utils/pb";
import { useMutation } from "@tanstack/react-query";
import { Save } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import ResourceForm from "./ResourceForm";
import { useContextResource } from "./utils";

type SpecComponent = React.ComponentType<{
  item: Resource;
  onUpdate: (item: Resource) => void;
}>;

export const ResourceEdit = (props: {
  item: Resource;
  specComponent: SpecComponent;
  dataComponent?: SpecComponent;
  noMetadata?: boolean;
  onUpdateDone?: (item: Resource) => void;
  noPostUpdateNavigation?: boolean;
  noPostUpdateToast?: boolean;
  readOnly?: boolean;
}) => {
  const navigate = useNavigate();
  const location = useLocation();

  const mutationUpdate = useMutation({
    mutationFn: async (arg: {
      req: Resource;
      yaml: string;
      isYAML: boolean;
    }) => {
      const resource = arg.isYAML ? resourceFromYAML(arg.yaml) : arg.req;
      if (!resource)
        throw new Error("Invalid YAML — could not parse resource.");
      const { response } = await updateResourcePB(resource);
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

  return (
    <ResourceForm
      item={props.item}
      specComponent={props.specComponent}
      dataComponent={props.dataComponent}
      noMetadata={props.noMetadata}
      readOnly={props.readOnly}
      requireDirty
      submitIcon={<Save size={13} strokeWidth={2.25} />}
      submitLabel="Save changes"
      submitPendingLabel="Saving…"
      isPending={mutationUpdate.isPending}
      isError={mutationUpdate.isError}
      errorLabel="Update failed — check the form and try again."
      onCancel={handleCancel}
      onSubmit={(req, yaml, isYAML) =>
        mutationUpdate.mutate({ req, yaml, isYAML })
      }
    />
  );
};

const ResourceEditPage = (props: {
  specComponent: SpecComponent;
  dataComponent?: SpecComponent;
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
          readOnly={props.readOnly || !!ctx.data.metadata?.isSystem}
        />
      )}
    </PageWrap>
  );
};

export default ResourceEditPage;
