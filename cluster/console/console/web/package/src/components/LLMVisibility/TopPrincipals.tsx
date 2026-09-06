import {
  Filter,
  ListTopPolicyRequest,
  ListTopServiceRequest,
  ListTopSessionRequest,
  ListTopUserRequest,
  Metric,
} from "@/apis/visibilityv1/llm/vllmv1";
import TopList from "@/components/TopList";
import { getClientVisibilityLLM } from "@/utils/client";
import { Resource } from "@/utils/pb";
import { Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { QueryState } from "./Primitives";
import { METRIC_ACCESSORS, metricAccessor } from "./utils";

type PrincipalKind = "user" | "session" | "service" | "policy";

const TopPrincipals = (props: {
  filter: Filter;
  kinds: PrincipalKind[];
  enabled?: boolean;
}) => {
  const [metric, setMetric] = React.useState<Metric>(Metric.REQUESTS);
  const accessor = metricAccessor(metric);
  const enabled = props.enabled ?? true;

  const qry = useQuery({
    queryKey: [
      "visibility",
      "llm",
      "topPrincipals",
      { filter: props.filter, kinds: props.kinds, metric },
    ],
    enabled,
    queryFn: async () => {
      const client = getClientVisibilityLLM();
      const ret: Record<string, { resource: Resource; count: number }[]> = {};

      for (const kind of props.kinds) {
        switch (kind) {
          case "user": {
            const { response } = await client.listTopUser(
              ListTopUserRequest.create({
                filter: props.filter,
                limit: 10,
                orderBy: metric,
                includeQuantiles: accessor.kind === "duration",
              }),
            );
            ret.user = response.items
              .filter((item) => item.user)
              .map((item) => ({
                resource: item.user as Resource,
                count: accessor.get(item.stats),
              }));
            break;
          }
          case "session": {
            const { response } = await client.listTopSession(
              ListTopSessionRequest.create({
                filter: props.filter,
                limit: 10,
                orderBy: metric,
                includeQuantiles: accessor.kind === "duration",
              }),
            );
            ret.session = response.items
              .filter((item) => item.session)
              .map((item) => ({
                resource: item.session as Resource,
                count: accessor.get(item.stats),
              }));
            break;
          }
          case "service": {
            const { response } = await client.listTopService(
              ListTopServiceRequest.create({
                filter: props.filter,
                limit: 10,
                orderBy: metric,
                includeQuantiles: accessor.kind === "duration",
              }),
            );
            ret.service = response.items
              .filter((item) => item.service)
              .map((item) => ({
                resource: item.service as Resource,
                count: accessor.get(item.stats),
              }));
            break;
          }
          case "policy": {
            const { response } = await client.listTopPolicy(
              ListTopPolicyRequest.create({
                filter: props.filter,
                limit: 10,
                orderBy: metric,
                includeQuantiles: accessor.kind === "duration",
              }),
            );
            ret.policy = response.items
              .filter((item) => item.policy)
              .map((item) => ({
                resource: item.policy as Resource,
                count: accessor.get(item.stats),
              }));
            break;
          }
        }
      }

      return ret;
    },
  });

  const titles: Record<PrincipalKind, string> = {
    user: "Top users",
    session: "Top sessions",
    service: "Top LLM Services",
    policy: "Top matched Policies",
  };

  const isEmpty =
    qry.isSuccess &&
    props.kinds.every((kind) => (qry.data?.[kind]?.length ?? 0) === 0);

  return (
    <div className="flex w-full flex-col gap-3">
      <Select
        size="xs"
        className="max-w-[220px]"
        label="Rank by"
        allowDeselect={false}
        value={String(metric)}
        data={METRIC_ACCESSORS.map((item) => ({
          value: String(item.metric),
          label: item.label,
        }))}
        onChange={(value) => value && setMetric(Number(value))}
      />
      <QueryState
        isLoading={enabled && qry.isPending}
        isError={qry.isError}
        isEmpty={enabled && isEmpty}
        minHeight={180}
      >
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {props.kinds.map((kind) => (
            <TopList
              key={kind}
              title={titles[kind]}
              items={qry.data?.[kind]}
            />
          ))}
        </div>
      </QueryState>
    </div>
  );
};

export default TopPrincipals;
