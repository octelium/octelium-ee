import { Session } from "@/apis/corev1/corev1";
import { GetOptions, ObjectReference } from "@/apis/metav1/metav1";
import { getType } from "@/pages/core/Session/List";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ResourceListLabel } from "../ResourceList";

const DoCardSession = ({ session }: { session: Session }) => {
  const userRef = session.status?.userRef;
  const deviceRef = session.status?.deviceRef;

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1.5 py-0.5">
      <ResourceListLabel label="Session" itemRef={getResourceRef(session)} />
      <ResourceListLabel label="Type">{getType(session)}</ResourceListLabel>
      {(userRef?.uid || userRef?.name) && (
        <ResourceListLabel label="User" itemRef={userRef} />
      )}
      {(deviceRef?.uid || deviceRef?.name) && (
        <ResourceListLabel label="Device" itemRef={deviceRef} />
      )}
    </div>
  );
};

const CardSession = ({ itemRef }: { itemRef: ObjectReference }) => {
  const refKey = itemRef.uid || itemRef.name;
  const qrySession = useQuery({
    queryKey: ["sessionCard", "session", refKey],
    queryFn: async () => {
      const { response } = await getClientCore().getSession(
        GetOptions.create({ uid: itemRef.uid, name: itemRef.name }),
      );
      return response;
    },
    enabled: !!refKey,
  });

  if (qrySession.data) return <DoCardSession session={qrySession.data} />;

  return (
    <div className="flex min-h-6 items-center">
      <ResourceListLabel label="Session">
        {itemRef.name || itemRef.uid || "Unknown"}
      </ResourceListLabel>
    </div>
  );
};

export default CardSession;
