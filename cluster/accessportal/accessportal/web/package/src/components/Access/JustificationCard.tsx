import { MessageSquareText } from "lucide-react";

import { Note, SectionCard } from "@/ui";
import { Tone } from "@/utils";

const JustificationCard = (props: {
  text?: string;
  title?: string;
  description?: string;
  emptyLabel?: string;
  emptyTone?: Tone;
}) => (
  <SectionCard
    title={props.title ?? "Justification"}
    description={props.description}
    icon={<MessageSquareText size={14} strokeWidth={2.25} />}
  >
    {props.text ? (
      <blockquote className="whitespace-pre-wrap border-l-2 border-slate-200 pl-3 text-body font-normal leading-relaxed text-slate-800 [overflow-wrap:anywhere]">
        {props.text}
      </blockquote>
    ) : (
      <Note tone={props.emptyTone}>
        {props.emptyLabel ?? "No justification was provided for this request."}
      </Note>
    )}
  </SectionCard>
);

export default JustificationCard;
