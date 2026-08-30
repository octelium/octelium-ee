import * as AccessP from "@/apis/accessv1/accessv1";
import * as CoreP from "@/apis/corev1/corev1";
import { ArrowRight, UserRound, Users } from "lucide-react";

import { Badge, Eyebrow, SectionCard, UserChip } from "@/ui";
import {
  requestSubjectName,
  requesterName,
  shortName,
  userTypeMeta,
} from "@/utils";

import { useSubjectUser } from "./hooks";

const Person = (props: {
  label?: string;
  name: string;
  fallbackLabel?: string;
}) => {
  const query = useSubjectUser(props.name);
  const user = query.data;
  const type = user ? userTypeMeta(user.type as CoreP.User_Spec_Type) : undefined;

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-2">
      {props.label && <Eyebrow>{props.label}</Eyebrow>}
      <UserChip
        src={user?.picURL}
        name={user?.displayName || shortName(props.name) || props.fallbackLabel || "—"}
        secondary={user?.email || props.name}
        badge={
          type ? (
            <Badge tone={type.tone} className="normal-case">
              {type.label}
            </Badge>
          ) : undefined
        }
      />
    </div>
  );
};

const PeopleCard = (props: { request: AccessP.Request }) => {
  const requester = requesterName(props.request);
  const subject = requestSubjectName(props.request);
  const onBehalf = !!subject && subject !== requester;

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
          <Users size={14} strokeWidth={2.4} />
        ) : (
          <UserRound size={14} strokeWidth={2.4} />
        )
      }
      tone={onBehalf ? "amber" : "blue"}
    >
      {onBehalf ? (
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
          <Person label="Requested by" name={requester} />
          <ArrowRight
            size={16}
            strokeWidth={2.4}
            className="hidden shrink-0 text-slate-300 sm:block"
          />
          <Person label="Access for" name={subject} />
        </div>
      ) : (
        <Person name={requester} />
      )}
    </SectionCard>
  );
};

export default PeopleCard;
