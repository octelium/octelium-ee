import * as AccessP from "@/apis/accessv1/accessv1";
import { ArrowRight, UserRound, Users } from "lucide-react";

import { Badge, Eyebrow, SectionCard, UserChip } from "@/ui";
import {
  isOnBehalfOf,
  isWorkloadUser,
  requestSubjectName,
  requesterName,
  shortName,
} from "@/utils";

import { useSubjectUser } from "./hooks";

const Person = (props: { label?: string; name: string }) => {
  const query = useSubjectUser(props.name);
  const user = query.data;

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-2">
      {props.label && <Eyebrow>{props.label}</Eyebrow>}
      <UserChip
        src={user?.picURL}
        name={user?.displayName || shortName(props.name) || "—"}
        secondary={user?.email || props.name}
        badge={isWorkloadUser(user?.type) ? <Badge>Workload</Badge> : undefined}
      />
    </div>
  );
};

const PeopleCard = (props: { request: AccessP.Request }) => {
  const requester = requesterName(props.request);
  const onBehalf = isOnBehalfOf(props.request);

  return (
    <SectionCard
      title={onBehalf ? "Requester and recipient" : "Requester"}
      description={
        onBehalf
          ? "This request was raised on behalf of another user"
          : "The user who raised and receives this access"
      }
      icon={
        onBehalf ? (
          <Users size={14} strokeWidth={2.25} />
        ) : (
          <UserRound size={14} strokeWidth={2.25} />
        )
      }
    >
      {onBehalf ? (
        <div className="flex flex-col gap-4 @md:flex-row @md:items-center">
          <Person label="Requested by" name={requester} />
          <ArrowRight
            size={16}
            strokeWidth={2.25}
            aria-hidden
            className="hidden shrink-0 text-slate-400 @md:block"
          />
          <Person label="Access for" name={requestSubjectName(props.request)} />
        </div>
      ) : (
        <Person name={requester} />
      )}
    </SectionCard>
  );
};

export default PeopleCard;
