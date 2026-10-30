import {
  hydrateLayouts,
  layoutStorageKey,
  loadLayouts,
  saveLayouts,
  setLayout,
  StoredLayout,
} from "@/features/dashboardLayout/slice";
import { useAppDispatch, useAppSelector } from "@/utils/hooks";
import { ActionIcon, Button, Drawer, Tooltip } from "@mantine/core";
import { Reorder, useDragControls } from "framer-motion";
import {
  ChevronDown,
  ChevronsDownUp,
  ChevronsUpDown,
  ChevronUp,
  EyeOff,
  GripVertical,
  Lock,
  LucideIcon,
  Plus,
} from "lucide-react";
import * as React from "react";
import { useLocation } from "react-router-dom";
import { showActionNotification } from "@/utils/notifications";
import { Deferred } from "./components";
import { WidgetContext } from "./widget";

export type DashboardWidget = {
  id: string;
  title: string;
  description: string;
  icon: LucideIcon;
  height?: number;
  eager?: boolean;
  locked?: boolean;
  render: () => React.ReactNode;
};

export type ResolvedLayout = {
  order: string[];
  hidden: string[];
  collapsed: string[];
};

type WidgetMeta = { id: string; locked?: boolean };

export const resolveLayout = (
  widgets: WidgetMeta[],
  stored?: StoredLayout,
): ResolvedLayout => {
  const ids = new Set(widgets.map((itm) => itm.id));
  const locked = new Set(
    widgets.filter((itm) => itm.locked).map((itm) => itm.id),
  );
  const order = [...new Set((stored?.order ?? []).filter((id) => ids.has(id)))];
  for (const itm of widgets) {
    if (!order.includes(itm.id)) order.push(itm.id);
  }
  const keep = (list?: string[]) => [
    ...new Set((list ?? []).filter((id) => ids.has(id) && !locked.has(id))),
  ];

  return {
    order,
    hidden: keep(stored?.hidden),
    collapsed: keep(stored?.collapsed),
  };
};

export const toStoredLayout = (
  widgets: WidgetMeta[],
  layout: ResolvedLayout,
): StoredLayout | undefined =>
  layout.hidden.length === 0 &&
  layout.collapsed.length === 0 &&
  layout.order.every((id, idx) => id === widgets[idx]?.id)
    ? undefined
    : {
        order: layout.order,
        hidden: layout.hidden,
        collapsed: layout.collapsed,
      };

export const useDashboardLayout = (page: string, widgets: WidgetMeta[]) => {
  const dispatch = useAppDispatch();
  const status = useAppSelector((state) => state.settings.status);
  const hydratedKey = useAppSelector((state) => state.dashboardLayout.key);
  const layouts = useAppSelector((state) => state.dashboardLayout.layouts);
  const uid = status?.user?.metadata?.uid;
  const key =
    status?.domain && uid ? layoutStorageKey(status.domain, uid) : undefined;

  React.useEffect(() => {
    if (key && key !== hydratedKey) {
      dispatch(hydrateLayouts({ key, layouts: loadLayouts(key) }));
    }
  }, [dispatch, key, hydratedKey]);

  const stored = key === hydratedKey ? layouts[page] : undefined;
  const layout = React.useMemo(
    () => resolveLayout(widgets, stored),
    [widgets, stored],
  );

  const save = (next: ResolvedLayout) => {
    const value = toStoredLayout(widgets, next);
    dispatch(setLayout({ page, layout: value }));
    if (key && key === hydratedKey) {
      const rest = Object.fromEntries(
        Object.entries(layouts).filter(([id]) => id !== page),
      );
      saveLayouts(key, value ? { ...rest, [page]: value } : rest);
    }
  };

  return { layout, save };
};

const WidgetIcon = (props: { icon: LucideIcon }) => {
  const Icon = props.icon;
  return (
    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
      <Icon size={13} strokeWidth={2.2} />
    </span>
  );
};

const CollapsedWidget = (props: {
  widget: DashboardWidget;
  onExpand: () => void;
}) => (
  <section
    aria-label={props.widget.title}
    className="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-2.5 sm:px-5"
  >
    <div className="flex min-w-0 items-center gap-3">
      <WidgetIcon icon={props.widget.icon} />
      <div className="min-w-0">
        <h2 className="truncate text-body font-semibold text-slate-800">
          {props.widget.title}
        </h2>
        <p className="truncate text-micro font-normal text-slate-500">
          {props.widget.description}
        </p>
      </div>
    </div>
    <Button
      size="compact-xs"
      variant="default"
      leftSection={<ChevronsUpDown size={12} strokeWidth={2.4} />}
      aria-label={`Expand ${props.widget.title}`}
      onClick={props.onExpand}
    >
      Expand
    </Button>
  </section>
);

const LayoutRow = (props: {
  widget: DashboardWidget;
  index: number;
  count: number;
  collapsed: boolean;
  onMove: (delta: number) => void;
  onToggleCollapsed: () => void;
  onHide: () => void;
}) => {
  const controls = useDragControls();
  const { widget } = props;

  return (
    <Reorder.Item
      as="li"
      value={widget.id}
      dragListener={false}
      dragControls={controls}
      className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white py-2 pl-1 pr-2"
    >
      <span
        aria-hidden="true"
        onPointerDown={(event) => controls.start(event)}
        className="flex h-7 w-5 shrink-0 cursor-grab touch-none items-center justify-center text-slate-400 transition-colors duration-150 hover:text-slate-700 active:cursor-grabbing"
      >
        <GripVertical size={14} />
      </span>
      <WidgetIcon icon={widget.icon} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs font-semibold text-slate-800">
          {widget.title}
          {props.collapsed && (
            <span className="ml-1.5 font-normal text-slate-500">
              · collapsed
            </span>
          )}
        </p>
        <p className="truncate text-micro font-normal text-slate-500">
          {widget.description}
        </p>
      </div>
      <ActionIcon
        variant="subtle"
        color="gray"
        size="sm"
        aria-label={`Move ${widget.title} up`}
        disabled={props.index === 0}
        onClick={() => props.onMove(-1)}
      >
        <ChevronUp size={14} />
      </ActionIcon>
      <ActionIcon
        variant="subtle"
        color="gray"
        size="sm"
        aria-label={`Move ${widget.title} down`}
        disabled={props.index === props.count - 1}
        onClick={() => props.onMove(1)}
      >
        <ChevronDown size={14} />
      </ActionIcon>
      {widget.locked ? (
        <Tooltip
          label="Always shown so that a problem cannot be hidden"
          withArrow
        >
          <span
            role="img"
            aria-label={`${widget.title} is always shown`}
            className="flex h-[22px] w-[52px] shrink-0 items-center justify-center text-slate-400"
          >
            <Lock size={13} />
          </span>
        </Tooltip>
      ) : (
        <>
          <Tooltip label={props.collapsed ? "Expand" : "Collapse"} withArrow>
            <ActionIcon
              variant={props.collapsed ? "light" : "subtle"}
              color="gray"
              size="sm"
              aria-label={`Collapse ${widget.title}`}
              aria-pressed={props.collapsed}
              onClick={props.onToggleCollapsed}
            >
              <ChevronsDownUp size={14} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Hide" withArrow>
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              aria-label={`Hide ${widget.title}`}
              onClick={props.onHide}
            >
              <EyeOff size={14} />
            </ActionIcon>
          </Tooltip>
        </>
      )}
    </Reorder.Item>
  );
};

const CustomizeDrawer = (props: {
  opened: boolean;
  onClose: () => void;
  widgets: DashboardWidget[];
  layout: ResolvedLayout;
  onSave: (layout: ResolvedLayout) => void;
}) => {
  const [draft, setDraft] = React.useState(props.layout);

  React.useEffect(() => {
    if (props.opened) setDraft(props.layout);
  }, [props.opened]);

  const byID = new Map(props.widgets.map((itm) => [itm.id, itm]));
  const visible = draft.order.filter(
    (id) => !draft.hidden.includes(id) && byID.has(id),
  );
  const hidden = draft.order.filter(
    (id) => draft.hidden.includes(id) && byID.has(id),
  );

  const setVisible = (next: string[]) =>
    setDraft((current) => ({
      ...current,
      order: [
        ...next,
        ...current.order.filter((id) => current.hidden.includes(id)),
      ],
    }));

  const move = (id: string, delta: number) => {
    const idx = visible.indexOf(id);
    const next = visible.filter((itm) => itm !== id);
    next.splice(idx + delta, 0, id);
    setVisible(next);
  };

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="min(440px, 100vw)"
      padding="md"
      title={
        <span className="text-sm font-semibold text-slate-800">
          Customize this page
        </span>
      }
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
    >
      <div className="flex flex-col gap-5">
        <p className="text-xs font-normal leading-5 text-slate-500">
          Drag the sections or use the arrows to reorder them. Collapsed and
          hidden sections load no data. The layout is saved in this browser.
        </p>

        <Reorder.Group
          as="ul"
          axis="y"
          values={visible}
          onReorder={setVisible}
          aria-label="Visible sections"
          className="flex flex-col gap-1.5"
        >
          {visible.map((id, idx) => (
            <LayoutRow
              key={id}
              widget={byID.get(id)!}
              index={idx}
              count={visible.length}
              collapsed={draft.collapsed.includes(id)}
              onMove={(delta) => move(id, delta)}
              onToggleCollapsed={() =>
                setDraft((current) => ({
                  ...current,
                  collapsed: current.collapsed.includes(id)
                    ? current.collapsed.filter((itm) => itm !== id)
                    : [...current.collapsed, id],
                }))
              }
              onHide={() =>
                setDraft((current) => ({
                  ...current,
                  hidden: [...current.hidden, id],
                  collapsed: current.collapsed.filter((itm) => itm !== id),
                }))
              }
            />
          ))}
        </Reorder.Group>

        {hidden.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Hidden sections
            </span>
            {hidden.map((id) => {
              const widget = byID.get(id)!;
              return (
                <div
                  key={id}
                  className="flex items-center gap-2 rounded-lg border border-dashed border-slate-200 bg-slate-50/60 px-2 py-2"
                >
                  <WidgetIcon icon={widget.icon} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-semibold text-slate-700">
                      {widget.title}
                    </p>
                    <p className="line-clamp-2 text-micro font-normal text-slate-500">
                      {widget.description}
                    </p>
                  </div>
                  <Button
                    size="compact-xs"
                    variant="default"
                    leftSection={<Plus size={12} strokeWidth={2.4} />}
                    aria-label={`Show ${widget.title}`}
                    onClick={() =>
                      setDraft((current) => ({
                        ...current,
                        hidden: current.hidden.filter((itm) => itm !== id),
                        order: [
                          ...current.order.filter((itm) => itm !== id),
                          id,
                        ],
                      }))
                    }
                  >
                    Add
                  </Button>
                </div>
              );
            })}
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 pt-4">
          <Button
            variant="subtle"
            color="gray"
            size="xs"
            onClick={() => setDraft(resolveLayout(props.widgets))}
          >
            Reset to default
          </Button>
          <div className="flex items-center gap-2">
            <Button variant="default" size="xs" onClick={props.onClose}>
              Cancel
            </Button>
            <Button size="xs" onClick={() => props.onSave(draft)}>
              Save
            </Button>
          </div>
        </div>
      </div>
    </Drawer>
  );
};

export const DashboardLayout = (props: {
  page: string;
  widgets: DashboardWidget[];
  customizing: boolean;
  onCustomizingChange: (value: boolean) => void;
}) => {
  const { widgets, onCustomizingChange } = props;
  const { layout, save } = useDashboardLayout(props.page, widgets);
  const { hash } = useLocation();
  const byID = new Map(widgets.map((itm) => [itm.id, itm]));

  React.useEffect(() => {
    if (!hash.startsWith("#section-")) return;
    const frame = requestAnimationFrame(() =>
      document.getElementById(hash.slice(1))?.scrollIntoView({
        block: "start",
      }),
    );
    return () => cancelAnimationFrame(frame);
  }, [hash]);

  const update = (next: ResolvedLayout, message?: string) => {
    const previous = layout;
    save(next);
    if (message) {
      showActionNotification(message, {
        label: "Undo",
        onClick: () => save(previous),
      });
    }
  };

  return (
    <>
      {layout.order
        .filter((id) => !layout.hidden.includes(id) && byID.has(id))
        .map((id) => {
          const widget = byID.get(id)!;
          const collapsed = layout.collapsed.includes(id);
          return (
            <WidgetContext.Provider
              key={id}
              value={{
                id,
                locked: !!widget.locked,
                collapse: () =>
                  update({ ...layout, collapsed: [...layout.collapsed, id] }),
                hide: () =>
                  update(
                    { ...layout, hidden: [...layout.hidden, id] },
                    `${widget.title} is hidden`,
                  ),
                customize: () => onCustomizingChange(true),
              }}
            >
              <div id={`section-${id}`} className="scroll-mt-20">
                {collapsed ? (
                  <CollapsedWidget
                    widget={widget}
                    onExpand={() =>
                      update({
                        ...layout,
                        collapsed: layout.collapsed.filter((itm) => itm !== id),
                      })
                    }
                  />
                ) : (
                  <Deferred eager={widget.eager} height={widget.height}>
                    {widget.render()}
                  </Deferred>
                )}
              </div>
            </WidgetContext.Provider>
          );
        })}

      {layout.hidden.length > 0 && (
        <p className="text-center text-micro font-normal text-slate-500">
          {layout.hidden.length === 1
            ? "1 section is hidden"
            : `${layout.hidden.length} sections are hidden`}{" "}
          ·{" "}
          <button
            type="button"
            onClick={() => onCustomizingChange(true)}
            className="cursor-pointer rounded font-semibold text-slate-700 outline-none underline-offset-2 hover:underline focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            Customize
          </button>
        </p>
      )}

      <CustomizeDrawer
        opened={props.customizing}
        onClose={() => onCustomizingChange(false)}
        widgets={widgets}
        layout={layout}
        onSave={(next) => {
          save(next);
          onCustomizingChange(false);
        }}
      />
    </>
  );
};
