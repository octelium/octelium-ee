import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import RpcError from "@/components/RpcError";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import { getClientCordiumManagement } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Alert, Button, Loader } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import Edit from "./Edit";

export default () => {
  const query = useQuery({
    queryKey: ["cordium", "clusterconfig"],
    queryFn: async () => getClientCordiumManagement().getClusterConfig({}),
  });

  if (query.isLoading)
    return (
      <div className="flex min-h-64 items-center justify-center">
        <Loader size="sm" color="gray" />
      </div>
    );

  if (query.isError)
    return (
      <div className="mx-auto mt-10 max-w-xl">
        <Alert
          color="red"
          title="Could not load the Cordium cluster configuration"
          icon={<AlertTriangle size={16} />}
        >
          <div className="space-y-3">
            <RpcError error={query.error} />
            <Button
              type="button"
              size="compact-sm"
              variant="outline"
              onClick={() => query.refetch()}
            >
              Retry
            </Button>
          </div>
        </Alert>
      </div>
    );

  if (!query.data?.response)
    return (
      <div className="py-16 text-center text-sm font-semibold text-slate-500">
        The Cordium cluster configuration is unavailable.
      </div>
    );

  return (
    <div className="min-w-0 [&_fieldset]:min-w-0">
      <ResourceEdit
        item={query.data.response}
        specComponent={({ item, onUpdate }) => (
          <Edit
            item={item as CordiumP.ClusterConfig}
            onUpdate={(next) => onUpdate(next)}
          />
        )}
        noPostUpdateNavigation
        noPostUpdateToast
        noMetadata
        onUpdateDone={() => {
          invalidateKey(["cordium", "clusterconfig"]);
          notifications.show({
            color: "green",
            message: "Cordium ClusterConfig successfully updated",
          });
        }}
      />
    </div>
  );
};
