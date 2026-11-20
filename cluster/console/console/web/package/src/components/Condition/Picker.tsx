import { twMerge } from "@/lib/utils";
import { ActionIcon, Modal, ScrollArea, TextInput } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import {
  Braces,
  Check,
  FileCode,
  Layers,
  Search,
  Sparkles,
  X,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { BLOCKS, CATEGORIES, type BlockDef } from "./blocks";
import {
  CATEGORY_META,
  RECIPE_ICONS,
  TONE_CHIP,
  blockIcon,
  type Tone,
} from "./meta";
import {
  blockNode,
  celNode,
  everyNode,
  groupNode,
  opaNode,
  type ConditionNode,
} from "./model";
import { RECIPES, matchesQuery, searchText, type Recipe } from "./recipes";

export type PickerScope = "request" | "basic";

type Entry = {
  id: string;
  section: string;
  title: string;
  description: string;
  icon: LucideIcon;
  tone: Tone;
  badge?: string;
  effect?: "ALLOW" | "DENY";
  text: string;
  recipe?: Recipe;
  build: () => ConditionNode[];
};

const CUSTOM: Entry[] = [
  {
    id: "custom.cel",
    section: "custom",
    title: "CEL expression",
    description: "Write any condition with the Common Expression Language",
    icon: Braces,
    tone: "slate",
    text: "cel expression custom code advanced",
    build: () => [celNode("")],
  },
  {
    id: "custom.group",
    section: "custom",
    title: "Group of conditions",
    description: "Nest conditions where any, all or none must match",
    icon: Layers,
    tone: "slate",
    text: "group nested and or nor any all none",
    build: () => [groupNode("any", [])],
  },
  {
    id: "custom.every",
    section: "custom",
    title: "Every request",
    description: "Always matches, e.g. for a catch-all rule",
    icon: Check,
    tone: "slate",
    text: "every request all always match any catch-all",
    build: () => [everyNode()],
  },
  {
    id: "custom.opa",
    section: "custom",
    title: "OPA policy",
    description: "An inline Open Policy Agent Rego policy",
    icon: FileCode,
    tone: "slate",
    text: "opa rego open policy agent",
    build: () => [opaNode("")],
  },
];

const recipeEntry = (recipe: Recipe): Entry => {
  const sample = recipe.build();
  const count = sample.kind === "group" ? sample.children.length : 1;
  const category = recipe.category === "general" ? "general" : recipe.category;
  return {
    id: `recipe.${recipe.id}`,
    section: "suggested",
    title: recipe.title,
    description: recipe.description,
    icon: RECIPE_ICONS[recipe.icon],
    tone: CATEGORY_META[category].tone,
    badge: count > 1 ? `${count} conditions` : undefined,
    effect: recipe.effect,
    text: searchText(recipe.title, recipe.description, recipe.keywords),
    recipe,
    build: () => [recipe.build()],
  };
};

const blockEntry = (block: BlockDef): Entry => ({
  id: `block.${block.id}`,
  section: block.category,
  title: block.title,
  description: block.description,
  icon: blockIcon(block.id, block.category),
  tone: CATEGORY_META[block.category].tone,
  badge: block.group,
  text: searchText(
    block.title,
    block.description,
    block.keywords,
    block.group,
    block.category,
  ),
  build: () => [blockNode(block.id)],
});

type Section = {
  id: string;
  label: string;
  icon: LucideIcon;
  description?: string;
};

const SECTIONS: Section[] = [
  {
    id: "suggested",
    label: "Common use cases",
    icon: Sparkles,
    description: "Ready-made conditions to adjust",
  },
  ...CATEGORIES.map((category) => ({
    id: category.id,
    label: category.label,
    icon: CATEGORY_META[category.id].icon,
    description: category.description,
  })),
  {
    id: "custom",
    label: "Custom",
    icon: Braces,
    description: "CEL, OPA and nested groups",
  },
];

const EntryCard = (props: {
  entry: Entry;
  onPick: () => void;
  active: boolean;
}) => {
  const { entry } = props;
  const Icon = entry.icon;
  return (
    <button
      type="button"
      onClick={props.onPick}
      data-active={props.active || undefined}
      className="group flex min-w-0 cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-white p-3 text-left shadow-card outline-none transition-[border-color,box-shadow,background-color] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400 data-[active]:border-slate-400 data-[active]:ring-1 data-[active]:ring-slate-300"
    >
      <span
        className={twMerge(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border",
          TONE_CHIP[entry.tone],
        )}
      >
        <Icon size={15} strokeWidth={2.2} />
      </span>
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
          <span className="text-xs font-semibold text-slate-800 group-hover:text-slate-950">
            {entry.title}
          </span>
          {entry.badge && (
            <span className="rounded bg-slate-100 px-1 py-px text-micro font-semibold text-slate-500">
              {entry.badge}
            </span>
          )}
        </span>
        <span className="text-micro leading-4 text-slate-500">
          {entry.description}
        </span>
        {entry.effect === "DENY" && (
          <span className="mt-0.5 text-micro font-semibold text-red-700">
            Typically for a Deny rule
          </span>
        )}
      </span>
    </button>
  );
};

const ConditionPicker = (props: {
  opened: boolean;
  scope: PickerScope;
  title?: string;
  onClose: () => void;
  onPick: (nodes: ConditionNode[], recipe?: Recipe) => void;
}) => {
  const [query, setQuery] = React.useState("");
  const [section, setSection] = React.useState(
    props.scope === "basic" ? "custom" : "suggested",
  );
  const mobile = useMediaQuery("(max-width: 40em)");
  const viewport = React.useRef<HTMLDivElement>(null);

  const entries = React.useMemo(
    () =>
      props.scope === "basic"
        ? CUSTOM
        : [...RECIPES.map(recipeEntry), ...BLOCKS.map(blockEntry), ...CUSTOM],
    [props.scope],
  );

  const sections =
    props.scope === "basic"
      ? SECTIONS.filter((item) => item.id === "custom")
      : SECTIONS;

  React.useEffect(() => {
    viewport.current?.scrollTo({ top: 0 });
  }, [section, query]);

  const searching = query.trim().length > 0;
  const visible = searching
    ? entries.filter((entry) => matchesQuery(entry.text, query))
    : entries.filter((entry) => entry.section === section);

  const grouped = sections
    .map((item) => ({
      section: item,
      items: visible.filter((entry) => entry.section === item.id),
    }))
    .filter((item) => item.items.length > 0);

  const pick = (entry: Entry) => {
    props.onPick(entry.build(), entry.recipe);
  };

  const current = sections.find((item) => item.id === section);

  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      fullScreen={!!mobile}
      size="min(1000px, calc(100vw - 32px))"
      padding={0}
      withCloseButton={false}
      radius={mobile ? 0 : "lg"}
      styles={{ body: { padding: 0 } }}
    >
      <div className="flex h-[100dvh] flex-col sm:h-[min(720px,calc(100dvh-64px))]">
        <header className="flex shrink-0 flex-col gap-3 border-b border-slate-200 px-4 pb-3 pt-4 sm:px-5">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <Modal.Title className="text-sm font-bold tracking-tight text-slate-900">
                {props.title ?? "Add a condition"}
              </Modal.Title>
              <p className="mt-0.5 text-xs text-slate-500">
                {props.scope === "basic"
                  ? "Choose how to express the condition."
                  : "Start from a common use case or combine building blocks. Every value can be changed afterwards."}
              </p>
            </div>
            <ActionIcon
              variant="subtle"
              color="gray"
              aria-label="Close"
              onClick={props.onClose}
            >
              <X size={16} />
            </ActionIcon>
          </div>
          {props.scope !== "basic" && (
            <TextInput
              data-autofocus
              aria-label="Search conditions"
              placeholder="Search: group, business hours, read-only, country, MFA…"
              leftSection={<Search size={14} />}
              value={query}
              onChange={(event) => setQuery(event.currentTarget.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && visible[0]) {
                  event.preventDefault();
                  pick(visible[0]);
                }
              }}
            />
          )}
          {props.scope !== "basic" && !searching && (
            <nav
              aria-label="Condition categories"
              className="-mx-4 flex gap-1.5 overflow-x-auto px-4 pb-0.5 [scrollbar-width:none] sm:hidden"
            >
              {sections.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  aria-pressed={section === item.id}
                  onClick={() => setSection(item.id)}
                  className={twMerge(
                    "inline-flex h-7 shrink-0 cursor-pointer items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium outline-none",
                    section === item.id
                      ? "border-slate-800 bg-slate-800 text-white"
                      : "border-slate-200 bg-white text-slate-600",
                  )}
                >
                  <item.icon size={12} />
                  {item.label}
                </button>
              ))}
            </nav>
          )}
        </header>

        <div className="flex min-h-0 flex-1">
          {props.scope !== "basic" && (
            <nav
              aria-label="Condition categories"
              className="hidden w-56 shrink-0 flex-col gap-0.5 overflow-y-auto border-r border-slate-200 bg-slate-50/60 p-2 sm:flex"
            >
              {sections.map((item) => {
                const active = !searching && section === item.id;
                const count = entries.filter(
                  (entry) => entry.section === item.id,
                ).length;
                return (
                  <button
                    key={item.id}
                    type="button"
                    aria-pressed={active}
                    onClick={() => {
                      setQuery("");
                      setSection(item.id);
                    }}
                    className={twMerge(
                      "flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-2 text-left text-xs font-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                      active
                        ? "bg-white text-slate-900 shadow-card ring-1 ring-slate-200"
                        : "text-slate-600 hover:bg-white/70 hover:text-slate-900",
                    )}
                  >
                    <item.icon
                      size={14}
                      className={active ? "text-slate-800" : "text-slate-400"}
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {item.label}
                    </span>
                    <span className="text-micro tabular-nums text-slate-400">
                      {count}
                    </span>
                  </button>
                );
              })}
            </nav>
          )}

          <ScrollArea
            className="min-w-0 flex-1"
            viewportRef={viewport}
            type="auto"
          >
            <div className="flex flex-col gap-5 p-4 sm:p-5">
              {!searching &&
                current?.description &&
                props.scope !== "basic" && (
                  <div className="flex items-center gap-2">
                    <current.icon size={15} className="text-slate-500" />
                    <h3 className="text-body font-semibold text-slate-800">
                      {current.label}
                    </h3>
                    <span className="text-xs text-slate-500">
                      · {current.description}
                    </span>
                  </div>
                )}
              {grouped.length === 0 && (
                <div className="flex flex-col items-center gap-2 px-4 py-12 text-center">
                  <Search size={20} className="text-slate-300" />
                  <p className="text-body font-semibold text-slate-700">
                    No match for “{query}”
                  </p>
                  <p className="max-w-sm text-xs text-slate-500">
                    Try other words, or write the condition yourself with a CEL
                    expression.
                  </p>
                  <button
                    type="button"
                    onClick={() => pick(CUSTOM[0])}
                    className="mt-1 cursor-pointer text-xs font-semibold text-blue-700 hover:text-blue-800"
                  >
                    Write a CEL expression
                  </button>
                </div>
              )}
              {grouped.map((group) => (
                <section
                  key={group.section.id}
                  className="flex flex-col gap-2.5"
                >
                  {searching && (
                    <h3 className="flex items-center gap-1.5 text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                      <group.section.icon size={12} />
                      {group.section.label}
                    </h3>
                  )}
                  <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
                    {group.items.map((entry) => (
                      <EntryCard
                        key={entry.id}
                        entry={entry}
                        active={searching && entry === visible[0]}
                        onPick={() => pick(entry)}
                      />
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </ScrollArea>
        </div>
      </div>
    </Modal>
  );
};

export default ConditionPicker;
