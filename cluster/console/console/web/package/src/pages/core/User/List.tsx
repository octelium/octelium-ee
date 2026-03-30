import {
  ListCredentialOptions,
  ListDeviceOptions,
  ListSessionOptions,
  ListUserOptions,
  User_Spec_Type,
  UserList,
} from "@/apis/corev1/corev1";

import { User } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { GetUserSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { ListAuthenticationLogRequest } from "@/apis/visibilityv1/visibilityv1";
import PieChart from "@/components/Charts/PieChart";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { setListOptFilter } from "@/features/settings/slice";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientCore, getClientVisibilityCore } from "@/utils/client";
import { useAppDispatch } from "@/utils/hooks";
import { getResourceRef, invalidateResourceListFromList } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import {
  Bot,
  LaptopMinimal,
  Mail,
  Shield,
  Terminal,
  User as UserIcon,
  Users,
} from "lucide-react";
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import { match } from "ts-pattern";

const getType = (svc: User) => {
  return match(svc.spec?.type)
    .with(User_Spec_Type.HUMAN, () => (
      <span className="flex items-center text-xs">
        <UserIcon size={14} />
        <span className="ml-1">Human</span>
      </span>
    ))
    .with(User_Spec_Type.WORKLOAD, () => (
      <span className="flex items-center">
        <Bot size={14} />
        <span className="ml-1">Workload</span>
      </span>
    ))
    .otherwise(() => <></>);
};

const ItemDetails = (props: { item: User; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: User }) => {
  const { item } = props;

  const qrySess = useQuery({
    queryKey: ["core.listSession", "usr", item.metadata!.name],
    queryFn: async () => {
      return await getClientCore().listSession(
        ListSessionOptions.create({ userRef: getResourceRef(item) })
      );
    },
  });

  const qryDev = useQuery({
    queryKey: ["core.listDevice", "usr", item.metadata!.name],
    queryFn: async () => {
      return await getClientCore().listDevice(
        ListDeviceOptions.create({ userRef: getResourceRef(item) })
      );
    },
  });

  const qryCred = useQuery({
    queryKey: ["core.listCredential", "usr", item.metadata!.name],
    queryFn: async () => {
      return await getClientCore().listCredential(
        ListCredentialOptions.create({ userRef: getResourceRef(item) })
      );
    },
  });

  const qryAuthn = useQuery({
    queryKey: ["core.listAuthenticator", "usr", item.metadata!.name],
    queryFn: async () => {
      return await getClientCore().listAuthenticator(
        ListAuthenticationLogRequest.create({ userRef: getResourceRef(item) })
      );
    },
  });

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>{getType(item)}</ResourceListLabel>

      {item.spec?.email && (
        <ResourceListLabel>
          <span className="flex items-center">
            <Mail size={14} />
            <span className="ml-1">{item.spec.email}</span>
          </span>
        </ResourceListLabel>
      )}

      {item.spec?.isDisabled && (
        <ResourceListLabel>
          <span className="flex items-center text-red-400">Disabled</span>
        </ResourceListLabel>
      )}

      {item.spec!.groups && item.spec!.groups.length > 0 && (
        <ResourceListLabel>
          <Users size={14} /> {item.spec?.groups.length} Groups
        </ResourceListLabel>
      )}

      {qrySess.isSuccess &&
        qrySess.data.response.listResponseMeta &&
        qrySess.data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/sessions`, {
              "userRef.name": item.metadata!.name,
            })}
          >
            <Terminal size={14} className="mr-1" />
            {qrySess.data.response.listResponseMeta.totalCount} Sessions
          </ResourceListLabel>
        )}

      {qryDev.isSuccess &&
        qryDev.data.response.listResponseMeta &&
        qryDev.data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/devices`, {
              "userRef.name": item.metadata!.name,
            })}
          >
            <LaptopMinimal size={14} className="mr-1" />
            {qryDev.data.response.listResponseMeta.totalCount} Devices
          </ResourceListLabel>
        )}

      {qryAuthn.isSuccess &&
        qryAuthn.data.response.listResponseMeta &&
        qryAuthn.data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/authenticators`, {
              "userRef.name": item.metadata!.name,
            })}
          >
            <LaptopMinimal size={14} className="mr-1" />
            {qryAuthn.data.response.listResponseMeta.totalCount} Authenticators
          </ResourceListLabel>
        )}

      {qryCred.isSuccess &&
        qryCred.data.response.listResponseMeta &&
        qryCred.data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/credentials`, {
              "userRef.name": item.metadata!.name,
            })}
          >
            <LaptopMinimal size={14} className="mr-1" />
            {qryCred.data.response.listResponseMeta.totalCount} Credentials
          </ResourceListLabel>
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
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: User }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

export const ListFilter = () => {
  const dispatch = useAppDispatch();

  React.useEffect(() => {
    invalidateResourceListFromList(
      UserList.create({ apiVersion: "core/v1", kind: "UserList" })
    );
    dispatch(
      setListOptFilter({
        listOptFilter: ListUserOptions.create({}),
      })
    );

    invalidateResourceListFromList(
      UserList.create({ apiVersion: "core/v1", kind: "UserList" })
    );
  }, []);

  return <></>;
};

const DoSummary = (props: { resp: GetUserSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/users`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalHuman}
          to={toURLWithQry(`/core/users`, {
            type: User_Spec_Type[User_Spec_Type.HUMAN],
          })}
          active={
            searchParams.get(`type`) === User_Spec_Type[User_Spec_Type.HUMAN]
          }
        >
          Humans
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWorkload}
          to={toURLWithQry(`/core/users`, {
            type: User_Spec_Type[User_Spec_Type.WORKLOAD],
          })}
          active={
            searchParams.get(`type`) === User_Spec_Type[User_Spec_Type.WORKLOAD]
          }
        >
          Workloads
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDisabled}
          to={toURLWithQry(`/core/users`, {
            isDisabled: "true",
          })}
          active={searchParams.get(`isPublic`) === "true"}
        >
          Disabled
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: {
  children?: (r: GetUserSummaryResponse) => React.ReactNode;
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "User"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getUserSummary({});

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
                { name: "Human", value: d.totalHuman },
                { name: "Workload", value: d.totalWorkload },
              ]}
            />
          )}
        </div>
      )}

      {props.children && <div>{props.children(qry.data)}</div>}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
