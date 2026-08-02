import { Service } from "@/apis/corev1/corev1";
import { GetOptions, ObjectReference } from "@/apis/metav1/metav1";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ResourceListLabel } from "../ResourceList";

const DoCardService = ({ service }: { service: Service }) => {
  const namespaceRef = service.status?.namespaceRef;

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1.5 py-0.5">
      <ResourceListLabel label="Service" itemRef={getResourceRef(service)} />
      {(namespaceRef?.uid || namespaceRef?.name) && (
        <ResourceListLabel label="Namespace" itemRef={namespaceRef} />
      )}
    </div>
  );
};

const CardService = ({ itemRef }: { itemRef: ObjectReference }) => {
  const refKey = itemRef.uid || itemRef.name;
  const qryService = useQuery({
    queryKey: ["serviceCard", "service", refKey],
    queryFn: async () => {
      const { response } = await getClientCore().getService(
        GetOptions.create({ uid: itemRef.uid, name: itemRef.name }),
      );
      return response;
    },
    enabled: !!refKey,
  });

  if (qryService.data) return <DoCardService service={qryService.data} />;

  return (
    <div className="flex min-h-6 items-center">
      <ResourceListLabel label="Service">
        {itemRef.name || itemRef.uid || "Unknown"}
      </ResourceListLabel>
    </div>
  );
};

export default CardService;
