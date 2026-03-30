import { Session } from "@/apis/corev1/corev1";
import { GetOptions, ObjectReference } from "@/apis/metav1/metav1";
import { DeviceTypeLabel } from "@/pages/core/Device/List";
import { getType } from "@/pages/core/Session/List";
import { getClientCore } from "@/utils/client";
import { getResourceRef, printUserWithEmail } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ResourceListLabel } from "../ResourceList";

const DoCardSession = (props: { session: Session }) => {
  const { session } = props;

  const qryUser = useQuery({
    queryKey: ["sessionCard", "user", session.status!.userRef!.uid],
    queryFn: async () => {
      const { response } = await getClientCore().getUser(
        GetOptions.create({
          uid: session.status!.userRef!.uid,
        })
      );

      return response;
    },
  });

  const qryDevice = useQuery({
    queryKey: ["sessionCard", "device", session.status!.deviceRef?.uid],
    queryFn: async () => {
      const { response } = await getClientCore().getDevice(
        GetOptions.create({
          uid: session.status?.deviceRef?.uid,
        })
      );

      return response;
    },
    enabled: !!session.status?.deviceRef?.uid,
  });

  const user = qryUser.data;
  const device = qryDevice.data;

  return (
    <>
      <ResourceListLabel
        label="Session"
        itemRef={getResourceRef(session)}
      ></ResourceListLabel>
      <ResourceListLabel>{getType(session)}</ResourceListLabel>
      {user && (
        <ResourceListLabel label="User" itemRef={getResourceRef(user)}>
          {printUserWithEmail(user)}
        </ResourceListLabel>
      )}

      {device && (
        <>
          <ResourceListLabel
            label="Device"
            itemRef={getResourceRef(device)}
          ></ResourceListLabel>

          <DeviceTypeLabel item={device} />
        </>
      )}
    </>
  );

  /*
  return (
    <div className="bg-gray-900 text-white py-2 px-4 shadow-lg rounded-lg inline-flex hover:shadow-2xl transition-all duration-500">
      <div className="flex items-center justify-center">
        {picURL.length > 0 && (
          <img
            src={picURL}
            className="w-[60px] h-[60px] rounded-full shadow-md mr-2"
          />
        )}
        <div>
          <div>
            <Link
              to={getResourcePath(user)}
              className="text-slate-100 hover:text-white duration-500 transition-all"
            >
              <div className="flex flex-col">
                <div className="flex flex-col">
                  {user.metadata!.displayName && (
                    <div className="text-sm">{user.metadata!.displayName}</div>
                  )}
                  <div className="flex items-center">
                    <div className="text-xs mt-1 text-slate-200">
                      {user.metadata!.name}
                    </div>
                    <Badge
                      className="mx-1"
                      variant="outline"
                      size="xs"
                      color="blue"
                    >
                      {match(user.spec!.type)
                        .with(CoreP.User_Spec_Type.HUMAN, () => "Human")
                        .with(CoreP.User_Spec_Type.WORKLOAD, () => "Workload")
                        .otherwise(() => "")}
                    </Badge>
                  </div>
                </div>
                {user.spec!.email && (
                  <div className="text-xs text-slate-200">
                    {user.spec!.email}
                  </div>
                )}
              </div>
            </Link>
          </div>
        </div>
        <div className="ml-2"></div>
      </div>
    </div>
  );
  */
};

const CardSession = (props: { itemRef: ObjectReference }) => {
  const qrySession = useQuery({
    queryKey: ["sessionCard", "session", props.itemRef.uid],
    queryFn: async () => {
      const { response } = await getClientCore().getSession(
        GetOptions.create({
          uid: props.itemRef.uid,
        })
      );

      return response;
    },
  });

  // return <></>;

  return (
    qrySession.data &&
    qrySession.isSuccess && <DoCardSession session={qrySession.data} />
  );
};

export default CardSession;
