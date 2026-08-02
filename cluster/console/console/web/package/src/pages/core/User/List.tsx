import {
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
import PieChart from "@/components/Charts/PieChart";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { setListOptFilter } from "@/features/settings/slice";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useAppDispatch } from "@/utils/hooks";
import { invalidateResourceListFromList } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import {
  Bot,
  Mail,
  User as UserIcon,
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
      UserList.create({ apiVersion: "core/v1", kind: "UserList" }),
    );
    dispatch(
      setListOptFilter({
        listOptFilter: ListUserOptions.create({}),
      }),
    );

    invalidateResourceListFromList(
      UserList.create({ apiVersion: "core/v1", kind: "UserList" }),
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
          active={searchParams.get(`isDisabled`) === "true"}
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
