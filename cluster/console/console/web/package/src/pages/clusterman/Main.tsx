import {
  ClusterConfig_Status_UpgradeRequest_State,
  UpgradeClusterRequest,
} from "@/apis/enterprisev1/enterprisev1";
import InfoItem from "@/components/InfoItem";
import TimeAgo from "@/components/TimeAgo";
import { onError } from "@/utils";
import { getClientCluster, getClientEnterprise } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Button, Modal, Text, TextInput } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation, useQuery } from "@tanstack/react-query";
import * as React from "react";
import ClipLoader from "react-spinners/ClipLoader";
import { toast } from "sonner";
import { match } from "ts-pattern";

import { Group, Switch } from "@mantine/core";

import {
  UpgradeClusterRequest_Request_Core,
  UpgradeClusterRequest_Request_PackageCordium,
  UpgradeClusterRequest_Request_PackageEnterprise,
} from "@/apis/enterprisev1/enterprisev1";
import { twMerge } from "tailwind-merge";

export default () => {
  const qry = useQuery({
    queryKey: ["clusterman", "main", "getCluster"],
    queryFn: async () => {
      return await getClientEnterprise().getClusterConfig({} as any);
    },
    refetchInterval: 15000,
  });

  if (!qry.isSuccess) {
    return <></>;
  }

  if (!qry.data) {
    return <></>;
  }

  const cluster = qry.data.response;

  return (
    <div className="w-full">
      <div className="w-full border-[1px] rounded-md border-gray-300 shadow-sm p-4">
        {cluster.status!.lastUpgradeRequests.length === 0 &&
          !cluster!.status!.upgradeRequest && (
            <Text size={"sm"} fw={"bold"} c="dimmed">
              No Prior Upgrades
            </Text>
          )}

        {cluster.status!.totalSuccessfulUpgrades > 0 && (
          <Text size="xs" fw={"bold"}>
            <InfoItem title="Successful Upgrades">
              {cluster.status!.totalSuccessfulUpgrades}
            </InfoItem>
          </Text>
        )}

        {cluster.status!.totalFailedUpgrades > 0 && (
          <Text size="xs" fw={"bold"}>
            <InfoItem title="Failed Upgrades">
              {cluster.status!.totalFailedUpgrades}
            </InfoItem>
          </Text>
        )}

        {cluster.status!.lastUpgradeRequests.length > 0 && (
          <Text size="xs" fw={"bold"}>
            <InfoItem title="Last Upgrade">
              <TimeAgo
                rfc3339={cluster.status!.lastUpgradeRequests[0].doneAt}
              />
            </InfoItem>

            <InfoItem title="State">
              {match(cluster.status!.lastUpgradeRequests[0].state)
                .with(
                  ClusterConfig_Status_UpgradeRequest_State.FAILED,
                  () => "Failed",
                )
                .with(
                  ClusterConfig_Status_UpgradeRequest_State.SUCCESS,
                  () => "Succeeded",
                )
                .otherwise(() => (
                  <></>
                ))}
            </InfoItem>

            {
              <InfoItem title="Version">
                {cluster.status!.lastUpgradeRequests[0].request?.core?.version}
              </InfoItem>
            }
          </Text>
        )}

        {cluster.status?.upgradeRequest && (
          <Text size="xs" fw={"bold"}>
            <InfoItem title="Upgrade Started">
              <TimeAgo rfc3339={cluster.status.upgradeRequest.createdAt} />
            </InfoItem>
            {cluster.status.upgradeRequest.doneAt && (
              <InfoItem title="Upgrade Done">
                <TimeAgo rfc3339={cluster.status.upgradeRequest.doneAt} />
              </InfoItem>
            )}
            {cluster.status.upgradeRequest.request?.core?.version && (
              <InfoItem title="Core Version">
                {cluster.status.upgradeRequest.request.core.version}
              </InfoItem>
            )}
            {cluster.status.upgradeRequest.request?.packageEnterprise
              ?.version && (
              <InfoItem title="Enterprise Version">
                {
                  cluster.status.upgradeRequest.request.packageEnterprise
                    .version
                }
              </InfoItem>
            )}
            {cluster.status.upgradeRequest.request?.packageCordium?.version && (
              <InfoItem title="Cordium Version">
                {cluster.status.upgradeRequest.request.packageCordium.version}
              </InfoItem>
            )}
            <InfoItem title="State">
              {match(cluster.status.upgradeRequest.state)
                .with(
                  ClusterConfig_Status_UpgradeRequest_State.UPGRADING,
                  () => (
                    <div className="flex items-center justify-center">
                      <ClipLoader color={"#666"} loading={true} size={20} />
                      <span className="ml-1">Upgrading</span>
                    </div>
                  ),
                )
                .with(
                  ClusterConfig_Status_UpgradeRequest_State.UPGRADE_REQUESTED,
                  () => (
                    <div className="flex items-center justify-center">
                      <span>Upgrade Request</span>
                    </div>
                  ),
                )
                .otherwise(() => (
                  <></>
                ))}
            </InfoItem>
          </Text>
        )}

        <div className="w-full mt-8">
          <UpgradeCluster />
        </div>
      </div>
    </div>
  );
};

const UpgradeCluster = () => {
  const [opened, { open, close }] = useDisclosure(false);
  let [active, setActive] = React.useState(false);

  let [req, setReq] = React.useState(
    UpgradeClusterRequest.create({
      request: {},
    }),
  );

  const mutationDelete = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().upgradeCluster(req);
      return response;
    },
    onSuccess: (r) => {
      toast.success("Upgrade started...");
      invalidateKey(["clusterman", "main", "getCluster"]);
    },
    onError,
  });

  return (
    <div>
      <div className="w-full my-8 flex items-center justify-end">
        <Button size={`lg`} onClick={open}>
          <span>Upgrade Cluster</span>
        </Button>
      </div>

      <Modal
        opened={opened}
        size={`xl`}
        onClose={() => {
          close();
          setActive(false);
        }}
        centered
      >
        <div className="font-bold text-xl mb-4">
          {`Are you sure that you want to Upgrade the Cluster?`}
        </div>

        <div className="w-full my-4">
          <Group className="mb-8 min-h-[40px]" grow>
            <Switch
              checked={!!req.request?.core}
              onChange={(event) => {
                const checked = event.currentTarget.checked;
                let r = UpgradeClusterRequest.clone(req);
                if (checked && !req.request?.core) {
                  r.request!.core = UpgradeClusterRequest_Request_Core.create();
                } else if (!checked && req.request?.core) {
                  r.request!.core = undefined;
                }
                setReq(r);
              }}
              label={`Upgrade the Cluster Core`}
            />
            {req.request && req.request.core && (
              <TextInput
                label="Version"
                placeholder="1.2.3"
                description={`Set the desired version (default is "latest")`}
                value={req.request.core.version}
                onChange={(v) => {
                  let r = UpgradeClusterRequest.clone(req);
                  r.request!.core!.version = v.target.value;
                  setReq(r);
                }}
              />
            )}
          </Group>

          <Group className="mb-8 min-h-[40px]" grow>
            <Switch
              checked={!!req.request?.packageEnterprise}
              onChange={(event) => {
                const checked = event.currentTarget.checked;
                let r = UpgradeClusterRequest.clone(req);
                if (checked && !req.request?.packageEnterprise) {
                  r.request!.packageEnterprise =
                    UpgradeClusterRequest_Request_PackageEnterprise.create();
                } else if (!checked && req.request?.packageEnterprise) {
                  r.request!.packageEnterprise = undefined;
                }
                setReq(r);
              }}
              label={`Upgrade the Cluster Enterprise Package`}
            />
            {req.request && req.request.packageEnterprise && (
              <TextInput
                label="Version"
                placeholder="1.2.3"
                description={`Set the desired version (default is "latest")`}
                value={req.request.packageEnterprise.version}
                onChange={(v) => {
                  let r = UpgradeClusterRequest.clone(req);
                  r.request!.packageEnterprise!.version = v.target.value;
                  setReq(r);
                }}
              />
            )}
          </Group>

          <Group className="mb-8 min-h-[40px]" grow>
            <Switch
              checked={!!req.request?.packageCordium}
              onChange={(event) => {
                const checked = event.currentTarget.checked;
                let r = UpgradeClusterRequest.clone(req);
                if (checked && !req.request?.packageCordium) {
                  r.request!.packageCordium =
                    UpgradeClusterRequest_Request_PackageCordium.create();
                } else if (!checked && req.request?.packageCordium) {
                  r.request!.packageCordium = undefined;
                }
                setReq(r);
              }}
              label={`Upgrade the Cordium Package`}
            />
            {req.request && req.request.packageCordium && (
              <TextInput
                label="Version"
                placeholder="1.2.3"
                description={`Set the desired version (default is "latest")`}
                value={req.request.packageCordium.version}
                onChange={(v) => {
                  let r = UpgradeClusterRequest.clone(req);
                  r.request!.packageCordium!.version = v.target.value;
                  setReq(r);
                }}
              />
            )}
          </Group>
        </div>

        <div className="flex items-center w-full">
          <div>
            <Switch
              checked={active}
              onChange={(event) => setActive(event.currentTarget.checked)}
              size="lg"
              label={`Yes, Upgrade the Cluster`}
            />
          </div>

          <div className="mt-4 flex justify-end items-center flex-1 ml-4">
            <Button
              variant="outline"
              size={`lg`}
              onClick={() => {
                setActive(false);
                close();
              }}
            >
              Cancel
            </Button>
            <Button
              className={twMerge(
                "ml-4  transition-all duration-500",
                active ? `shadow-lg` : undefined,
              )}
              size={`lg`}
              disabled={!active}
              loading={mutationDelete.isPending}
              onClick={() => {
                mutationDelete.mutate();
                close();
              }}
              autoFocus
            >
              Upgrade
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
