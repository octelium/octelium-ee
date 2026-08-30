import { MessageSquareQuote } from "lucide-react";

import { Note, SectionCard } from "@/ui";

const JustificationCard = (props: {
  text?: string;
  title?: string;
  description?: string;
  emptyLabel?: string;
}) => (
  <SectionCard
    title={props.title ?? "Justification"}
    description={props.description}
    icon={<MessageSquareQuote size={14} strokeWidth={2.4} />}
    tone={props.text ? "blue" : "slate"}
  >
    {props.text ? (
      <blockquote className="border-l-2 border-slate-200 pl-3 text-[0.82rem] font-medium leading-relaxed whitespace-pre-wrap text-slate-700">
        {props.text}
      </blockquote>
    ) : (
      <Note tone="amber">
        {props.emptyLabel ?? "No justification was provided for this request."}
      </Note>
    )}
  </SectionCard>
);

export default JustificationCard;
