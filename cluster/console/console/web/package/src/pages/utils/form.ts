import {
  cloneResource,
  getPB,
  Resource,
  resourceFromJSON,
  resourceToJSON,
} from "@/utils/pb";
import { useForm } from "@mantine/form";

interface useResourceFormProps {
  item: Resource;
  onUpdate: (arg: any) => void;
  transformValues?: (arg: any) => void;
}
export const useResourceForm = (props: useResourceFormProps) => {
  const { item, onUpdate } = props;

  return useForm({
    mode: "uncontrolled",
    initialValues: JSON.parse(resourceToJSON(item)),
    onValuesChange: (v) => {
      const res = resourceFromJSON(JSON.stringify(v));
      if (!res) {
        return;
      }

      if (
        //@ts-ignore
        !getPB(item)[`${item.kind}_Spec`].equals(item.spec, res.spec) ||
        item.kind.endsWith(`Secret`)
      ) {
        onUpdate(cloneResource(res));
      }
    },
    // transformValues: props.transformValues,
  });
};
