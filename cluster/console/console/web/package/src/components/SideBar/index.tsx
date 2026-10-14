import {
  findSection,
  navSections,
  pageAgent,
  pageOverview,
  pageSettings,
  type NavItem,
} from "@/components/Navigation/catalog";
import { twMerge } from "@/lib/utils";
import { Menu, SegmentedControl, Tooltip } from "@mantine/core";
import { useLocalStorage, useReducedMotion } from "@mantine/hooks";
import {
  Check,
  ChevronDown,
  Layers,
  ListTree,
  PanelLeftClose,
  PanelLeftOpen,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { functionGroups, type FunctionGroup } from "./functions";
import classes from "./styles.module.css";

const MenuButton = React.forwardRef<
  HTMLButtonElement,
  React.ComponentPropsWithoutRef<"button"> & { "data-expanded"?: boolean }
>((props, ref) => (
  <Tooltip
    label={props.title ?? ""}
    disabled={!props.title}
    opened={props["data-expanded"] ? false : undefined}
    position="right"
    openDelay={200}
    events={{ hover: !!props.title, focus: !!props.title, touch: false }}
    withArrow
  >
    <button
      {...props}
      title={undefined}
      ref={ref}
      onKeyDown={(event) => {
        props.onKeyDown?.(event);
        if (
          event.defaultPrevented ||
          !["ArrowDown", "ArrowUp"].includes(event.key)
        )
          return;
        event.preventDefault();
        const target = event.currentTarget;
        const last = event.key === "ArrowUp";
        if (!target.dataset.expanded) target.click();
        requestAnimationFrame(() => {
          const menu = document.getElementById(
            target.getAttribute("aria-controls") ?? "",
          );
          const items = menu?.querySelectorAll<HTMLElement>("[data-menu-item]");
          items?.[last ? items.length - 1 : 0]?.focus();
        });
      }}
    />
  </Tooltip>
));

const isCurrent = (pathname: string, url: string) =>
  pathname === url || (url !== "/" && pathname.startsWith(`${url}/`));

const NavigationLink = (props: {
  item: NavItem;
  collapsed: boolean;
  exact?: boolean;
}) => {
  const { pathname } = useLocation();
  const { item, collapsed } = props;
  const selected = props.exact
    ? pathname === item.url
    : isCurrent(pathname, item.url);

  return (
    <Tooltip
      label={item.title}
      disabled={!collapsed}
      position="right"
      openDelay={200}
      events={{ hover: collapsed, focus: collapsed, touch: false }}
      withArrow
    >
      <Link
        to={item.url}
        aria-label={item.title}
        aria-current={selected ? "page" : undefined}
        className={twMerge(
          classes.row,
          "flex h-9 shrink-0 items-center gap-2.5 rounded-lg px-2.5 text-body outline-none transition-[background-color,color,box-shadow] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:h-11",
          selected
            ? "bg-white font-semibold text-slate-900 shadow-card ring-1 ring-slate-200"
            : "font-normal text-slate-600 hover:bg-slate-200/50 hover:text-slate-900",
        )}
      >
        <item.icon
          size={16}
          strokeWidth={selected ? 2.25 : 1.75}
          className="shrink-0"
          aria-hidden
        />
        <span className={classes.label}>{item.title}</span>
      </Link>
    </Tooltip>
  );
};

const WorkspacePicker = (props: { collapsed: boolean }) => {
  const { pathname } = useLocation();
  const section = findSection(pathname);
  const navigate = useNavigate();
  const Icon = section?.icon ?? Layers;
  const reducedMotion = useReducedMotion();

  return (
    <Menu
      withInitialFocusPlaceholder={false}
      position={props.collapsed ? "right-start" : "bottom-start"}
      width={248}
      withinPortal
      loop
      closeOnItemClick
      transitionProps={{ transition: "pop", duration: reducedMotion ? 0 : 150 }}
    >
      <Menu.Target>
        <MenuButton
          type="button"
          aria-label={`API workspace ${section?.label ?? "Choose workspace"}`}
          title={
            props.collapsed
              ? `API workspace: ${section?.label ?? "Choose workspace"}`
              : undefined
          }
          className={twMerge(
            classes.row,
            "group flex h-10 w-full shrink-0 items-center gap-2.5 rounded-lg border border-slate-200 bg-white px-2.5 text-left text-body text-slate-800 shadow-card outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:h-11",
          )}
        >
          <Icon size={16} className="shrink-0 text-slate-500" aria-hidden />
          <span
            className={twMerge(classes.label, "min-w-0 flex-1 font-semibold")}
          >
            {section?.label ?? "Choose workspace"}
          </span>
          {!props.collapsed && (
            <ChevronDown
              size={14}
              className="shrink-0 text-slate-500"
              aria-hidden
            />
          )}
        </MenuButton>
      </Menu.Target>
      <Menu.Dropdown style={{ maxWidth: "calc(100vw - 24px)" }}>
        <Menu.Label>API workspaces</Menu.Label>
        {navSections.map((workspace) => (
          <Menu.RadioItem
            key={workspace.prefix}
            value={workspace.prefix}
            checked={workspace === section}
            onClick={() => navigate(workspace.defaultPath)}
            closeMenuOnClick
            checkIcon={<Check size={14} aria-hidden />}
            className={
              workspace === section
                ? "!bg-slate-100 !text-slate-900"
                : undefined
            }
          >
            <span className="flex items-center gap-2">
              <workspace.icon size={15} aria-hidden />
              <span>{workspace.label}</span>
            </span>
          </Menu.RadioItem>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
};

const Group = (props: { group: FunctionGroup; collapsed: boolean }) => {
  const { pathname } = useLocation();
  const { group, collapsed } = props;
  const active = group.items.some((item) => isCurrent(pathname, item.url));
  const [state, setState] = React.useState({ pathname, opened: active });
  const opened = active && state.pathname !== pathname ? true : state.opened;
  const id = React.useId();

  if (collapsed) {
    return (
      <Menu
        withInitialFocusPlaceholder={false}
        position="right-start"
        width={248}
        withinPortal
        loop
      >
        <Menu.Target>
          <MenuButton
            type="button"
            aria-label={group.label}
            data-active={active || undefined}
            title={group.label}
            className={twMerge(
              "flex h-9 w-full items-center justify-center rounded-lg text-slate-600 outline-none transition-[background-color,color,box-shadow] duration-150 hover:bg-slate-200/50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:h-11",
              active &&
                "bg-white text-slate-900 shadow-card ring-1 ring-slate-200",
            )}
          >
            <group.icon size={16} aria-hidden />
          </MenuButton>
        </Menu.Target>
        <Menu.Dropdown
          style={{ maxHeight: "calc(100dvh - 88px)", overflowY: "auto" }}
        >
          <Menu.Label>{group.label}</Menu.Label>
          {group.items.map((item) => (
            <Menu.Item
              key={item.url}
              component={Link}
              to={item.url}
              aria-current={isCurrent(pathname, item.url) ? "page" : undefined}
              leftSection={<item.icon size={15} aria-hidden />}
              className={
                isCurrent(pathname, item.url)
                  ? "!bg-slate-100 !text-slate-900"
                  : undefined
              }
            >
              {item.title}
            </Menu.Item>
          ))}
        </Menu.Dropdown>
      </Menu>
    );
  }

  return (
    <section aria-label={group.label}>
      <button
        type="button"
        aria-expanded={opened}
        aria-controls={id}
        onClick={() => setState({ pathname, opened: !opened })}
        className="flex h-9 w-full items-center gap-2 rounded-lg px-2.5 text-xs font-semibold text-slate-500 outline-none transition-colors hover:bg-slate-200/40 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:h-11"
      >
        <group.icon size={14} className="shrink-0" aria-hidden />
        <span className="min-w-0 flex-1 text-left">{group.label}</span>
        <ChevronDown
          size={13}
          aria-hidden
          className={twMerge(
            "shrink-0 transition-transform duration-150",
            !opened && "-rotate-90",
          )}
        />
        {active && !opened && (
          <span className="h-1 w-1 rounded-full bg-slate-700" />
        )}
      </button>
      <ul id={id} hidden={!opened} className="space-y-px">
        {group.items.map((item) => (
          <li key={item.url}>
            <NavigationLink item={item} collapsed={collapsed} />
          </li>
        ))}
      </ul>
    </section>
  );
};

export default function Sidebar(props: {
  collapsed?: boolean;
  onToggle?: () => void;
  onClose?: () => void;
}) {
  const collapsed = props.collapsed ?? false;
  const { pathname } = useLocation();
  const section = findSection(pathname);
  const [storedMode, setMode] = useLocalStorage<"api" | "functions">({
    key: "octelium-console-navigation-mode",
    defaultValue: "api",
    getInitialValueInEffect: false,
    deserialize: (value) => (value === '"functions"' ? "functions" : "api"),
  });
  const mode = storedMode === "functions" ? "functions" : "api";
  const reducedMotion = useReducedMotion();
  const navRef = React.useRef<HTMLElement>(null);

  React.useEffect(() => {
    const nav = navRef.current;
    if (!nav) return;
    const reveal = () => {
      const selected = nav.querySelector<HTMLElement>(
        '[aria-current="page"], [data-active]',
      );
      if (!selected || selected.getClientRects().length === 0) return;
      const bounds = nav.getBoundingClientRect();
      const item = selected.getBoundingClientRect();
      if (item.top < bounds.top)
        nav.scrollTop += Math.floor(item.top - bounds.top);
      else if (item.bottom > bounds.bottom)
        nav.scrollTop += Math.ceil(item.bottom - bounds.bottom);
    };
    reveal();
    const observer = new ResizeObserver(reveal);
    observer.observe(nav);
    return () => observer.disconnect();
  }, [pathname, mode, collapsed]);

  const toggleLabel = props.onClose
    ? "Close navigation"
    : collapsed
      ? "Expand sidebar"
      : "Collapse sidebar";
  const ToggleIcon = props.onClose
    ? X
    : collapsed
      ? PanelLeftOpen
      : PanelLeftClose;

  return (
    <div
      className={twMerge(
        classes.sidebar,
        "flex h-full min-h-0 w-full flex-col",
      )}
      data-collapsed={collapsed || undefined}
    >
      <div
        className={twMerge(
          classes.row,
          "flex h-12 shrink-0 items-center justify-between gap-2 px-3",
        )}
      >
        <span
          className={twMerge(
            classes.label,
            "text-xs font-semibold text-slate-500",
          )}
        >
          Console
        </span>
        {(props.onToggle || props.onClose) && (
          <Tooltip label={toggleLabel} position="right" withArrow>
            <button
              type="button"
              aria-label={toggleLabel}
              aria-expanded={props.onClose ? undefined : !collapsed}
              onClick={props.onClose ?? props.onToggle}
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-500 outline-none transition-colors hover:bg-slate-200/60 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:h-11 [@media(pointer:coarse)]:w-11"
            >
              <ToggleIcon size={17} aria-hidden />
            </button>
          </Tooltip>
        )}
      </div>

      <div className="shrink-0 space-y-1 px-2 pb-3">
        <NavigationLink item={pageOverview} collapsed={collapsed} />
        <NavigationLink item={pageAgent} collapsed={collapsed} />
      </div>

      <div className="shrink-0 space-y-2 border-t border-slate-200/80 px-2 py-3">
        {collapsed ? (
          <Menu
            withInitialFocusPlaceholder={false}
            position="right-start"
            width={192}
            withinPortal
            loop
          >
            <Menu.Target>
              <MenuButton
                type="button"
                aria-label={`Navigation view: ${mode === "api" ? "API" : "Functions"}`}
                title={`Navigation view: ${mode === "api" ? "API" : "Functions"}`}
                className="flex h-9 w-full items-center justify-center rounded-lg text-slate-500 outline-none transition-colors hover:bg-slate-200/50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                {mode === "api" ? (
                  <Layers size={16} aria-hidden />
                ) : (
                  <ListTree size={16} aria-hidden />
                )}
              </MenuButton>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>Navigation view</Menu.Label>
              {(["api", "functions"] as const).map((value) => (
                <Menu.RadioItem
                  key={value}
                  value={value}
                  checked={mode === value}
                  onChange={() => setMode(value)}
                  closeMenuOnClick
                  checkIcon={<Check size={14} aria-hidden />}
                >
                  {value === "api" ? "API" : "Functions"}
                </Menu.RadioItem>
              ))}
            </Menu.Dropdown>
          </Menu>
        ) : (
          <SegmentedControl
            aria-label="Navigation view"
            value={mode}
            onChange={setMode}
            data={[
              { value: "api", label: "API" },
              { value: "functions", label: "Functions" },
            ]}
            fullWidth
            size="xs"
            withItemsBorders={false}
            transitionDuration={reducedMotion ? 0 : 150}
          />
        )}
        {mode === "api" && <WorkspacePicker collapsed={collapsed} />}
      </div>

      <nav
        ref={navRef}
        aria-label="Console"
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-2 pb-3 [scrollbar-width:thin]"
      >
        {mode === "api" ? (
          <>
            {!collapsed && (
              <p className="px-2.5 pb-2 pt-1 text-xs font-semibold text-slate-500">
                {section ? "Resources" : "API workspaces"}
              </p>
            )}
            <ul className="space-y-px">
              {(section
                ? [
                    ...(section.items.some(
                      (item) => item.url === section.defaultPath,
                    )
                      ? []
                      : [
                          {
                            title: `${section.label} overview`,
                            url: section.defaultPath,
                            icon: pageOverview.icon,
                          },
                        ]),
                    ...section.items,
                  ]
                : navSections.map((workspace) => ({
                    title: workspace.label,
                    url: workspace.defaultPath,
                    icon: workspace.icon,
                  }))
              ).map((item) => (
                <li key={item.url}>
                  <NavigationLink
                    item={item}
                    collapsed={collapsed}
                    exact={item.url === section?.defaultPath}
                  />
                </li>
              ))}
            </ul>
          </>
        ) : (
          <div className="space-y-px">
            {functionGroups.map((group) => (
              <Group key={group.id} group={group} collapsed={collapsed} />
            ))}
          </div>
        )}
      </nav>

      <div className="shrink-0 border-t border-slate-200/80 p-2">
        <NavigationLink item={pageSettings} collapsed={collapsed} />
      </div>
    </div>
  );
}
