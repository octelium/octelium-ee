import { AgentHeaderButton } from "@/components/AgentDock/AskButtons";
import {
  resolveTrail,
  type Crumb,
  type NavItem,
} from "@/components/Navigation/catalog";
import { Menu, Tooltip } from "@mantine/core";
import {
  Check,
  ChevronDown,
  ChevronRight,
  Ellipsis,
  House,
} from "lucide-react";
import * as React from "react";
import { Link, useLocation } from "react-router-dom";
import { twMerge } from "@/lib/utils";
import { usePageHeaderSetters } from "./context";

const isWithin = (pathname: string, url: string) =>
  url === "/"
    ? pathname === "/"
    : pathname === url || pathname.startsWith(`${url}/`);

const ChildrenMenu = (props: {
  label: string;
  items: NavItem[];
  views?: NavItem[];
  pathname: string;
  trigger: React.ReactElement;
}) => (
  <Menu
    position="bottom-start"
    offset={6}
    width={236}
    withinPortal
    transitionProps={{ transition: "pop-top-left", duration: 150 }}
  >
    <Menu.Target>{props.trigger}</Menu.Target>
    <Menu.Dropdown className="max-h-[min(420px,70vh)] overflow-y-auto">
      <Menu.Label>{props.label}</Menu.Label>
      {props.items.map((item) => {
        const active = isWithin(props.pathname, item.url);
        return (
          <Menu.Item
            key={item.url}
            component={Link}
            to={item.url}
            leftSection={<item.icon size={14} />}
            rightSection={
              active ? <Check size={13} strokeWidth={2.75} /> : undefined
            }
            className={active ? "!bg-slate-100" : undefined}
            aria-current={active ? "page" : undefined}
          >
            {item.title}
          </Menu.Item>
        );
      })}
      {props.views && props.views.length > 0 && (
        <>
          <Menu.Divider />
          <Menu.Label>Views</Menu.Label>
          {props.views.map((view) => (
            <Menu.Item
              key={view.url}
              component={Link}
              to={view.url}
              leftSection={<view.icon size={14} />}
            >
              {view.title}
            </Menu.Item>
          ))}
        </>
      )}
    </Menu.Dropdown>
  </Menu>
);

const Separator = (props: { crumb: Crumb; pathname: string }) => {
  const { crumb } = props;
  if (!crumb.children) {
    return (
      <ChevronRight
        size={13}
        strokeWidth={2.25}
        aria-hidden
        className="mx-0.5 shrink-0 text-slate-400"
      />
    );
  }
  return (
    <ChildrenMenu
      label={crumb.label}
      items={crumb.children}
      views={crumb.views}
      pathname={props.pathname}
      trigger={
        <button
          type="button"
          aria-label={`Pages in ${crumb.label}`}
          className="group flex h-6 w-5 shrink-0 cursor-pointer items-center justify-center rounded-md text-slate-400 outline-none transition-colors duration-150 hover:bg-slate-200/70 hover:text-slate-700 focus-visible:ring-2 focus-visible:ring-slate-400 aria-expanded:bg-slate-200/70 aria-expanded:text-slate-700"
        >
          <ChevronRight
            size={13}
            strokeWidth={2.25}
            className="transition-transform duration-200 group-aria-expanded:rotate-90"
          />
        </button>
      }
    />
  );
};

const crumbLinkClass =
  "flex h-7 min-w-0 items-center rounded-md px-1.5 text-slate-500 outline-none transition-colors duration-150 hover:bg-slate-200/60 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400";

const Title = (props: { crumb: Crumb; pathname: string }) => {
  const { crumb } = props;
  const Icon = crumb.icon;
  return (
    <span className="flex min-w-0 items-center gap-1">
      <h1 className="flex min-w-0 items-center gap-2 px-1.5 text-[15px] font-semibold leading-6 tracking-[-0.01em] text-slate-900">
        {Icon && (
          <Icon
            size={15}
            strokeWidth={2.25}
            className="shrink-0 text-slate-500"
          />
        )}
        <span className="truncate" title={crumb.label}>
          {crumb.label}
        </span>
      </h1>
      {crumb.children && (
        <ChildrenMenu
          label={crumb.label}
          items={crumb.children}
          views={crumb.views}
          pathname={props.pathname}
          trigger={
            <button
              type="button"
              aria-label={`Pages in ${crumb.label}`}
              className="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-md text-slate-500 outline-none transition-colors duration-150 hover:bg-slate-200/70 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400 aria-expanded:bg-slate-200/70 aria-expanded:text-slate-800"
            >
              <ChevronDown size={14} strokeWidth={2.25} />
            </button>
          }
        />
      )}
    </span>
  );
};

const Breadcrumbs = (props: {
  trail: Crumb[];
  pathname: string;
  addonRef: (element: HTMLElement | null) => void;
}) => {
  const { trail, pathname } = props;
  const last = trail.length - 1;
  const collapsed = trail.slice(1, Math.max(1, last - 1));

  return (
    <nav aria-label="Breadcrumb" className="flex min-w-0 flex-1 items-center">
      <ol className="flex min-w-0 items-center">
        {trail.map((crumb, idx) => {
          const isLast = idx === last;
          const isCollapsed = idx > 0 && idx < last - 1;
          return (
            <React.Fragment key={crumb.path}>
              {idx === 1 && collapsed.length > 0 && (
                <li className="flex shrink-0 items-center md:hidden">
                  <Menu position="bottom-start" offset={6} withinPortal>
                    <Menu.Target>
                      <button
                        type="button"
                        aria-label="Show the parent pages"
                        className="flex h-7 w-7 cursor-pointer items-center justify-center rounded-md text-slate-500 outline-none hover:bg-slate-200/60 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
                      >
                        <Ellipsis size={15} />
                      </button>
                    </Menu.Target>
                    <Menu.Dropdown>
                      {collapsed.map((itm) => (
                        <Menu.Item
                          key={itm.path}
                          component={Link}
                          to={itm.path}
                          leftSection={
                            itm.icon ? <itm.icon size={14} /> : undefined
                          }
                        >
                          {itm.label}
                        </Menu.Item>
                      ))}
                    </Menu.Dropdown>
                  </Menu>
                  <ChevronRight
                    size={13}
                    strokeWidth={2.25}
                    aria-hidden
                    className="mx-0.5 shrink-0 text-slate-400"
                  />
                </li>
              )}
              <li
                aria-current={isLast ? "page" : undefined}
                className={twMerge(
                  "flex items-center",
                  isLast ? "min-w-0" : "shrink-0",
                  isCollapsed && "hidden md:flex",
                )}
              >
                {isLast ? (
                  <Title crumb={crumb} pathname={pathname} />
                ) : idx === 0 ? (
                  <Tooltip label={crumb.label} withArrow openDelay={400}>
                    <Link
                      to={crumb.path}
                      aria-label={crumb.label}
                      className={twMerge(
                        crumbLinkClass,
                        "w-7 justify-center px-0",
                      )}
                    >
                      <House size={15} strokeWidth={2.1} />
                    </Link>
                  </Tooltip>
                ) : (
                  <Link
                    to={crumb.path}
                    className={`text-body font-medium ${crumbLinkClass}`}
                  >
                    <span className="max-w-[16rem] truncate">
                      {crumb.label}
                    </span>
                  </Link>
                )}
                {!isLast && <Separator crumb={crumb} pathname={pathname} />}
              </li>
            </React.Fragment>
          );
        })}
      </ol>
      <div
        ref={props.addonRef}
        className="ml-1.5 flex shrink-0 items-center gap-1.5 empty:hidden"
      />
    </nav>
  );
};

const PageHeader = () => {
  const { pathname } = useLocation();
  const { setActions, setAddon } = usePageHeaderSetters();
  const rootRef = React.useRef<HTMLDivElement>(null);
  const trail = React.useMemo(() => resolveTrail(pathname), [pathname]);

  React.useLayoutEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const root = document.documentElement;
    const update = () =>
      root.style.setProperty("--page-header-height", `${el.offsetHeight}px`);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => {
      observer.disconnect();
      root.style.removeProperty("--page-header-height");
    };
  }, []);

  return (
    <div
      ref={rootRef}
      className="relative z-30 -mx-4 -mt-4 border-b border-slate-200 bg-canvas sm:sticky sm:top-[var(--app-shell-header-offset,60px)]"
    >
      <div className="mx-auto flex min-h-12 w-full max-w-[calc(var(--page-max-width)+2rem)] flex-wrap items-center gap-x-3 gap-y-1.5 px-4 py-2">
        <Breadcrumbs trail={trail} pathname={pathname} addonRef={setAddon} />
        <div
          ref={setActions}
          className="order-3 -mx-4 flex w-[calc(100%+2rem)] items-center gap-2 overflow-x-auto px-4 py-0.5 [scrollbar-width:none] empty:hidden sm:order-2 sm:mx-0 sm:w-auto sm:flex-wrap sm:justify-end sm:overflow-visible sm:px-0"
        />
        <div className="order-2 flex shrink-0 empty:hidden sm:order-3">
          <AgentHeaderButton />
        </div>
      </div>
    </div>
  );
};

export default PageHeader;
