import { getClientVisibilityCore } from "@/utils/client";
import * as React from "react";

import { Service_Spec_Mode } from "@/apis/corev1/corev1";
import { useQuery } from "@tanstack/react-query";

import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { Service } from "@/apis/corev1/corev1";

import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import { getDomain } from "@/utils";
import { BiLinkExternal } from "react-icons/bi";
import { Link, useSearchParams } from "react-router-dom";

import { Button, Collapse } from "@mantine/core";

import { GetServiceSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import PieChart from "@/components/Charts/PieChart";
import ResourceYAML from "@/components/ResourceYAML";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import {
  getServiceHostname,
  getServicePrivateFQDN,
  getServicePublicFQDN,
} from "@/utils/octelium";
import { getResourcePath, printServiceMode } from "@/utils/pb";
import { Lock, Shield } from "lucide-react";

export const getType = (svc: Service): string => {
  return printServiceMode(svc.spec!.mode);
};

const ItemDetails = (props: { item: Service; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return (
    <div>
      {md.description && (
        <InfoItem title="Description">{md.description}</InfoItem>
      )}
      <InfoItem title="Private FQDN">
        <CopyText value={getServicePrivateFQDN(item, props.domain)} />
      </InfoItem>
      {item.spec?.isPublic && (
        <InfoItem title="Public FQDN">
          <CopyText value={getServicePublicFQDN(item, props.domain)} />
        </InfoItem>
      )}

      {item.status?.primaryHostname &&
        item.status?.primaryHostname.length > 0 && (
          <InfoItem title="Primary Hostname">
            <CopyText value={item.status.primaryHostname} />
          </InfoItem>
        )}
      {item.status?.addresses && item.status.addresses.length > 0 && (
        <InfoItem title="Private Addresses">
          <div className="flex flex-col">
            {item.status?.addresses.map((x) => (
              <span className="w-full" key={x.dualStackIP?.ipv4}>
                <CopyText value={x.dualStackIP?.ipv4} />
              </span>
            ))}
          </div>
        </InfoItem>
      )}
    </div>
  );
};

const Item = (props: { item: Service }) => {
  const { item } = props;
  const domain = getDomain();

  const md = item.metadata!;

  let [showDetails, setShowDetails] = React.useState(false);

  return (
    <div
      className="font-semibold w-full"
      onMouseEnter={() => {
        setShowDetails(true);
      }}
      onMouseLeave={() => {
        setShowDetails(false);
      }}
    >
      <div className="flex">
        <div className="flex flex-col flex-1">
          <div className="flex items-center font-bold">
            <ResourceYAML item={item} size="xs" />
            <Link to={getResourcePath(item)}>
              <span className="text-gray-800 mr-2 flex flex-row">
                <CopyText value={getServiceHostname(item)} />
              </span>
              {md.displayName && (
                <span className="text-gray-600">{md.displayName}</span>
              )}
            </Link>
          </div>

          <Collapse in={showDetails}>
            <ItemDetails item={item} domain={domain} />
          </Collapse>
        </div>
        <div className="flex items-start justify-center">
          {item.spec?.isPublic && item.spec.mode === Service_Spec_Mode.WEB && (
            <Button
              component={"a"}
              size="sm"
              href={`https://${getServicePublicFQDN(item, domain)}`}
              target="_blank"
            >
              <span className="px-1">Visit</span>
              <BiLinkExternal />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};

export const LabelComponent = (props: { item: Service }) => {
  const { item } = props;
  const domain = getDomain();

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel
        label="Mode"
        to={toURLWithQry(`/core/services`, {
          mode: Service_Spec_Mode[item.spec!.mode],
        })}
      >
        {getType(item)}
      </ResourceListLabel>
      <ResourceListLabel
        itemRef={item.status!.namespaceRef}
      ></ResourceListLabel>

      {item.status?.port && item.status.port > 0 && (
        <ResourceListLabel label="Port">{item.status.port}</ResourceListLabel>
      )}

      {item.spec?.isTLS && (
        <ResourceListLabel>
          <span className="flex items-center">
            <Lock size={14} />
            <span>TLS</span>
          </span>
        </ResourceListLabel>
      )}
      {item.spec?.isPublic && <ResourceListLabel>Public</ResourceListLabel>}
      {item.spec?.isAnonymous && (
        <ResourceListLabel>Anonymous Access</ResourceListLabel>
      )}
      {item.spec?.config?.type.oneofKind === `ssh` &&
        item.spec.config.type.ssh.eSSHMode && (
          <ResourceListLabel>Embedded SSH</ResourceListLabel>
        )}
      {item.spec?.authorization &&
        item.spec?.authorization.policies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.policies.length} Policies
          </ResourceListLabel>
        )}
      {item.spec?.authorization &&
        item.spec?.authorization.inlinePolicies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.inlinePolicies.length} Inline Policies
          </ResourceListLabel>
        )}
      <ResourceListLabel itemRef={item.status!.regionRef}></ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Service }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetServiceSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/core/services">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalHTTP}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.HTTP],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.HTTP]
          }
        >
          HTTP
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalTCP}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.TCP],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.TCP]
          }
        >
          TCP
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSSH}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.SSH],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.SSH]
          }
        >
          SSH
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWeb}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.WEB],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.WEB]
          }
        >
          Web
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalKubernetes}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.KUBERNETES],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.KUBERNETES]
          }
        >
          Kubernetes
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPostgres}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.POSTGRES],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.POSTGRES]
          }
        >
          PostgreSQL
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalMysql}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.MYSQL],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.MYSQL]
          }
        >
          MySQL
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalGRPC}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.GRPC],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.GRPC]
          }
        >
          gRPC
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDNS}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.DNS],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.DNS]
          }
        >
          DNS
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalUDP}
          to={toURLWithQry(`/core/services`, {
            mode: Service_Spec_Mode[Service_Spec_Mode.UDP],
          })}
          active={
            searchParams.get(`mode`) ===
            Service_Spec_Mode[Service_Spec_Mode.UDP]
          }
        >
          UDP
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPublic}
          to={toURLWithQry(`/core/services`, {
            isPublic: "true",
          })}
          active={searchParams.get(`isPublic`) === "true"}
        >
          Public
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalAnonymous}
          to={toURLWithQry(`/core/services`, {
            isAnonymous: "true",
          })}
          active={searchParams.get(`isAnonymous`) === "true"}
        >
          Anonymous
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: {
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Service"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getServiceSummary(
        {}
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const d = qry.data;

  return (
    <div>
      {d.totalNumber > 0 && (
        <div>
          <DoSummary resp={qry.data} />
          {props.pieMain && (
            <PieChart
              data={[
                { name: "HTTP", value: d.totalHTTP },
                { name: "Web", value: d.totalWeb },
                { name: "SSH", value: d.totalSSH },
                { name: "TCP", value: d.totalTCP },
                { name: "UDP", value: d.totalUDP },
                { name: "Kubernetes", value: d.totalKubernetes },
                { name: "MySQL", value: d.totalMysql },
                { name: "PostgreSQL", value: d.totalPostgres },
                { name: "DNS", value: d.totalDNS },
                { name: "gRPC", value: d.totalGRPC },
              ]}
            />
          )}
        </div>
      )}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
