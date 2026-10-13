import { isModKey } from "@/components/AgentDock/context";
import {
  kindIcon,
  navSections,
  pageAgent,
  pageOverview,
  pageSettings,
  sectionItems,
} from "@/components/Navigation/catalog";
import { ActionIcon } from "@mantine/core";
import { Search } from "lucide-react";
import * as React from "react";

const Palette = React.lazy(() => import("./Palette"));

const isMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

const pages = [
  pageOverview,
  pageAgent,
  pageSettings,
  ...sectionItems,
  ...navSections.flatMap((section) =>
    [
      ...section.items.filter((item) => item.url !== section.defaultPath),
      ...(section.views ?? []),
    ].map((item) => ({ ...item, subtitle: section.label })),
  ),
];

const shortcuts = [pageOverview, ...sectionItems];

const iconFor = (path: string) =>
  kindIcon(path) ?? pages.find((page) => page.url === path)?.icon;

const PaletteContext = React.createContext<() => void>(() => {});

export const CommandPaletteProvider = (props: {
  children: React.ReactNode;
}) => {
  const [opened, setOpened] = React.useState(false);
  const [loaded, setLoaded] = React.useState(false);

  const open = React.useCallback(() => {
    setLoaded(true);
    setOpened(true);
  }, []);

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        isModKey(event) &&
        !event.shiftKey &&
        !event.altKey &&
        event.key.toLowerCase() === "k"
      ) {
        event.preventDefault();
        setLoaded(true);
        setOpened((value) => !value);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  return (
    <PaletteContext.Provider value={open}>
      {props.children}
      {loaded && (
        <React.Suspense fallback={null}>
          <Palette
            opened={opened}
            onClose={() => setOpened(false)}
            pages={pages}
            shortcuts={shortcuts}
            iconFor={iconFor}
          />
        </React.Suspense>
      )}
    </PaletteContext.Provider>
  );
};

export const CommandPaletteTrigger = () => {
  const open = React.useContext(PaletteContext);
  const shortcut = isMac ? "Meta+K" : "Control+K";

  return (
    <>
      <button
        type="button"
        onClick={open}
        aria-keyshortcuts={shortcut}
        className="group hidden h-9 w-full max-w-[26rem] cursor-pointer items-center gap-2.5 rounded-lg border border-slate-200 bg-white/70 pl-3 pr-1.5 text-slate-500 outline-none transition-[border-color,background-color,box-shadow,color] duration-150 hover:border-slate-300 hover:bg-white hover:text-slate-700 hover:shadow-card focus-visible:ring-2 focus-visible:ring-slate-400 md:flex"
      >
        <Search size={14} strokeWidth={2.25} className="shrink-0" />
        <span className="min-w-0 flex-1 truncate text-left text-body font-normal">
          Search or jump to…
        </span>
        <kbd className="text-micro flex h-5 shrink-0 items-center gap-0.5 rounded-md border border-slate-200 bg-slate-50 px-1.5 font-semibold text-slate-400 transition-colors duration-150 group-hover:text-slate-500">
          {isMac ? "⌘" : "Ctrl"} K
        </kbd>
      </button>
      <ActionIcon
        variant="subtle"
        color="gray"
        size="lg"
        aria-label="Search or jump to"
        aria-keyshortcuts={shortcut}
        onClick={open}
        className="md:hidden"
      >
        <Search size={17} strokeWidth={2.25} />
      </ActionIcon>
    </>
  );
};
