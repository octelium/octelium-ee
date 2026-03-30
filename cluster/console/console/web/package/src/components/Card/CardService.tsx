import { Service } from "@/apis/corev1/corev1";
import { GetOptions, ObjectReference } from "@/apis/metav1/metav1";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ResourceListLabel } from "../ResourceList";

const DoCardService = (props: { service: Service }) => {
  const { service } = props;

  return (
    <>
      <ResourceListLabel
        label="Service"
        itemRef={getResourceRef(service)}
      ></ResourceListLabel>

      <ResourceListLabel
        label="Namespace"
        itemRef={service.status!.namespaceRef}
      ></ResourceListLabel>
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

const CardService = (props: { itemRef: ObjectReference }) => {
  const qryService = useQuery({
    queryKey: ["serviceCard", "service", props.itemRef.uid],
    queryFn: async () => {
      const { response } = await getClientCore().getService(
        GetOptions.create({
          uid: props.itemRef.uid,
        })
      );

      return response;
    },
  });

  // return <></>;

  return (
    qryService.data &&
    qryService.isSuccess && <DoCardService service={qryService.data} />
  );
};

export default CardService;
