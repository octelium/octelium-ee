import Section, { SectionProps } from "../Section";

const ItemMessage = (props: SectionProps & { title: string }) => (
  <Section {...props} />
);

export default ItemMessage;
