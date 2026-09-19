import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { match } from "ts-pattern";

export const getOriginLabel = (type?: AccessC.Origin_Type): string =>
  match(type)
    .with(AccessC.Origin_Type.API, () => "Cluster API")
    .with(AccessC.Origin_Type.INTEGRATION, () => "Integration")
    .with(AccessC.Origin_Type.SYSTEM, () => "Cluster")
    .otherwise(() => "Unset");

export const hasOrigin = (origin?: AccessC.Origin): boolean =>
  !!origin && origin.type !== AccessC.Origin_Type.TYPE_UNSET;

const refOf = (
  itemRef: ObjectReference | undefined,
  apiVersion: string,
  kind: string,
): ObjectReference | undefined =>
  itemRef?.name || itemRef?.uid
    ? ObjectReference.create({
        ...itemRef,
        apiVersion: itemRef.apiVersion || apiVersion,
        kind: itemRef.kind || kind,
      })
    : undefined;

export const OriginValue = (props: { origin: AccessC.Origin }) => {
  const { origin } = props;
  const integrationRef = refOf(
    origin.integrationRef,
    "access/v1",
    "Integration",
  );
  const sessionRef = refOf(origin.sessionRef, "core/v1", "Session");

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <Label
        tone={
          origin.type === AccessC.Origin_Type.INTEGRATION ? "info" : "neutral"
        }
        size="sm"
      >
        {getOriginLabel(origin.type)}
      </Label>
      {integrationRef && <ResourceListLabel itemRef={integrationRef} />}
      {sessionRef && <ResourceListLabel itemRef={sessionRef} />}
      {origin.externalActorID && (
        <ResourceListLabel label="External actor">
          <CopyText value={origin.externalActorID} />
        </ResourceListLabel>
      )}
      {origin.externalEventID && (
        <ResourceListLabel label="Provider event">
          <CopyText value={origin.externalEventID} />
        </ResourceListLabel>
      )}
    </div>
  );
};
