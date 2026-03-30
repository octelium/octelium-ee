import { ClusterConfig_Spec_Collector_Pipeline_Type } from "@/apis/enterprisev1/enterprisev1";

import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import EditItem from "@/components/EditItem";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import ResourceYAML from "@/components/ResourceYAML";
import { getClientEnterprise } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import {
  CloseButton,
  Group,
  NumberInput,
  Select,
  TextInput,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

import ItemMessage from "@/components/ItemMessage";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { strToNum } from "@/utils/convert";
import { useState } from "react";
import { toast } from "sonner";

const Edit = (props: {
  item: EnterpriseP.ClusterConfig;
  onUpdate: (item: EnterpriseP.ClusterConfig) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = useState(EnterpriseP.ClusterConfig.clone(item));
  const updateReq = () => {
    setReq(EnterpriseP.ClusterConfig.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <div className="w-full my-12">
        <ResourceYAML item={req} />
        <div className="w-full">
          <EditItem
            title="Scaler"
            description="Scale your different Cluster components"
            onUnset={() => {
              req.spec!.scaler = undefined;

              updateReq();
            }}
            obj={req.spec!.scaler}
            onSet={() => {
              if (!req.spec!.scaler) {
                req.spec!.scaler = EnterpriseP.ClusterConfig_Spec_Scaler.create(
                  {
                    collector: {},
                    ingress: {},
                    octovigil: {},
                  },
                );
              }
              updateReq();
            }}
          >
            {req.spec!.scaler && (
              <div>
                <Group grow>
                  <NumberInput
                    label="Collector Replicas"
                    description="Set the replicas of the OTEL Collector"
                    min={0}
                    max={32}
                    value={req.spec!.scaler!.collector!.replicas}
                    onChange={(v) => {
                      req.spec!.scaler!.collector!.replicas = strToNum(v);
                      updateReq();
                    }}
                  />

                  <NumberInput
                    label="Ingress Replicas"
                    description="Set the replicas of the Envoy ingress"
                    min={0}
                    max={32}
                    value={req.spec!.scaler!.ingress!.replicas}
                    onChange={(v) => {
                      req.spec!.scaler!.ingress!.replicas = strToNum(v);
                      updateReq();
                    }}
                  />

                  <NumberInput
                    label="Octovigil Replicas"
                    description="Set the replicas for Octovigil"
                    min={0}
                    max={32}
                    value={req.spec!.scaler!.octovigil!.replicas}
                    onChange={(v) => {
                      req.spec!.scaler!.octovigil!.replicas = strToNum(v);
                      updateReq();
                    }}
                  />
                </Group>
              </div>
            )}
          </EditItem>

          <EditItem
            title="Collector"
            description="Set your Collector pipelines and exporters"
            onUnset={() => {
              req.spec!.collector = undefined;

              updateReq();
            }}
            obj={req.spec!.collector}
            onSet={() => {
              if (!req.spec!.collector) {
                req.spec!.collector =
                  EnterpriseP.ClusterConfig_Spec_Collector.create();
                updateReq();
              }
            }}
          >
            {req.spec!.collector && (
              <div>
                <ItemMessage
                  title="Pipelines"
                  obj={req!.spec!.collector!.pipelines}
                  isList
                  onSet={() => {
                    req.spec!.collector!.pipelines = [
                      EnterpriseP.ClusterConfig_Spec_Collector_Pipeline.create(),
                    ];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.collector!.pipelines.push(
                      EnterpriseP.ClusterConfig_Spec_Collector_Pipeline.create(),
                    );
                    updateReq();
                  }}
                >
                  {req!.spec!.collector!.pipelines?.map(
                    (x: any, idx: number) => (
                      <div className="mb-4" key={`${idx}`}>
                        <div className="flex">
                          <CloseButton
                            size={"sm"}
                            variant="subtle"
                            onClick={() => {
                              req.spec!.collector!.pipelines.splice(idx, 1);
                              updateReq();
                            }}
                          ></CloseButton>
                          <div className="flex-1 ml-2">
                            <Group grow>
                              <TextInput
                                required
                                label="Name"
                                description="Define a unique name for the pipeline"
                                placeholder="logs-1"
                                value={req.spec!.collector!.pipelines[idx].name}
                                onChange={(v) => {
                                  req.spec!.collector!.pipelines[idx].name =
                                    v.target.value;
                                  updateReq();
                                }}
                              />

                              <Select
                                label="Type"
                                required
                                description="The pipeline can be either a LOGS or METRICS pipeline"
                                data={[
                                  {
                                    label: "Logs",
                                    value:
                                      ClusterConfig_Spec_Collector_Pipeline_Type[
                                        ClusterConfig_Spec_Collector_Pipeline_Type
                                          .LOGS
                                      ],
                                  },
                                  {
                                    label: "Metrics",
                                    value:
                                      ClusterConfig_Spec_Collector_Pipeline_Type[
                                        ClusterConfig_Spec_Collector_Pipeline_Type
                                          .METRICS
                                      ],
                                  },
                                ]}
                                value={
                                  ClusterConfig_Spec_Collector_Pipeline_Type[
                                    req.spec!.collector!.pipelines[idx].type
                                  ]
                                }
                                onChange={(v) => {
                                  req.spec!.collector!.pipelines[idx].type =
                                    ClusterConfig_Spec_Collector_Pipeline_Type[
                                      v as "METRICS"
                                    ];
                                  updateReq();
                                }}
                              />

                              <SelectResourceMultiple
                                api="enterprise"
                                kind="CollectorExporter"
                                label="Collector Exporters"
                                description="Select one or more Collector Exporters for this pipeline"
                                defaultValue={
                                  req.spec!.collector!.pipelines[idx].exporters
                                }
                                clearable
                                onChange={(v) => {
                                  req.spec!.collector!.pipelines[
                                    idx
                                  ].exporters =
                                    v?.map((x) => x.metadata!.name) ?? [];
                                  updateReq();
                                }}
                              />
                            </Group>
                          </div>
                        </div>
                      </div>
                    ),
                  )}
                </ItemMessage>
              </div>
            )}
          </EditItem>

          <EditItem
            title="Certificate"
            description="Set Certificate-specific configs"
            onUnset={() => {
              req.spec!.certificate = undefined;

              updateReq();
            }}
            obj={req.spec!.certificate}
            onSet={() => {
              if (!req.spec!.certificate) {
                req.spec!.certificate =
                  EnterpriseP.ClusterConfig_Spec_Certificate.create();
              }
              updateReq();
            }}
          >
            {req.spec!.certificate && (
              <div>
                <Group grow>
                  <Select
                    label="Mode"
                    required
                    description="Set the Certificate mode to MANAGED OR MANUAL"
                    data={[
                      {
                        label: "Managed",
                        value:
                          EnterpriseP.Certificate_Spec_Mode[
                            EnterpriseP.Certificate_Spec_Mode.MANAGED
                          ],
                      },
                      {
                        label: "Manual",
                        value:
                          EnterpriseP.Certificate_Spec_Mode[
                            EnterpriseP.Certificate_Spec_Mode.MANUAL
                          ],
                      },
                    ]}
                    defaultValue={
                      EnterpriseP.Certificate_Spec_Mode[
                        req.spec!.certificate!.defaultMode
                      ] ??
                      EnterpriseP.Certificate_Spec_Mode[
                        EnterpriseP.Certificate_Spec_Mode.MANAGED
                      ]
                    }
                    onChange={(v) => {
                      req.spec!.certificate!.defaultMode =
                        EnterpriseP.Certificate_Spec_Mode[v as "MANAGED"];
                      updateReq();
                    }}
                  />
                </Group>
              </div>
            )}
          </EditItem>
        </div>
      </div>
    </div>
  );
};

export default () => {
  const { isSuccess, isLoading, data } = useQuery({
    queryKey: ["enterprise", "clusterconfig"],
    queryFn: async () => {
      return await getClientEnterprise().getClusterConfig({});
    },
  });

  if (!isSuccess) {
    return <></>;
  }

  if (!data) {
    return <></>;
  }

  return (
    <div>
      {data && data.response && (
        <ResourceEdit
          item={data.response}
          // @ts-ignore
          specComponent={Edit}
          noPostUpdateNavigation
          noPostUpdateToast
          noMetadata
          onUpdateDone={(v) => {
            invalidateKey(["enterprise", "clusterconfig"]);
            toast.success("ClusterConfig successfully updated");
          }}
        />
      )}
    </div>
  );
};

/*
export default () => {
  const { isSuccess, isLoading, data } = useQuery({
    queryKey: ["enterprise", "clusterconfig"],
    queryFn: async () => {
      return await getClientEnterprise().getClusterConfig({});
    },
  });

  if (!isSuccess) {
    return <></>;
  }

  if (!data) {
    return <></>;
  }

  return (
    <div>
      {data && data.response && (
        <ResourceEdit
          item={data.response}
          // @ts-ignore
          specComponent={Edit}
          noPostUpdateNavigation
          noPostUpdateToast
          noMetadata
          onUpdateDone={(v) => {
            invalidateKey(["enterprise", "clusterconfig"]);
          }}
        />
      )}
    </div>
  );
};

const DNSProvider = () => {
  const { isSuccess, isLoading, data } = useQuery({
    queryKey: ["enterprise", "DNSProvider", "default"],
    queryFn: async () => {
      return await getClientEnterprise().getDNSProvider(
        GetOptions.create({ name: "default" }),
      );
    },
  });

  if (!isSuccess) {
    return <></>;
  }

  if (!data) {
    return <></>;
  }

  return (
    <div>
      <EditDNSProvider item={data.response} onUpdate={(itm) => {}} />
    </div>
  );
};
*/
