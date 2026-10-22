import { twMerge } from "@/lib/utils";
import { Combobox, useCombobox } from "@mantine/core";
import { Check, ChevronsUpDown, Search } from "lucide-react";
import * as React from "react";
import {
  CATEGORY_LABEL,
  ExporterCategory,
  ExporterKind,
  ExporterTile,
  getProvider,
  PROVIDERS,
  SignalBadges,
} from "./providers";

const matches = (query: string, text: string) => {
  const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
  const haystack = text.toLowerCase();
  return terms.every((term) => haystack.includes(term));
};

const DestinationPicker = (props: {
  value?: ExporterKind;
  onChange: (kind: ExporterKind) => void;
}) => {
  const [query, setQuery] = React.useState("");
  const combobox = useCombobox({
    onDropdownClose: () => {
      combobox.resetSelectedOption();
      setQuery("");
    },
    onDropdownOpen: () => {
      combobox.focusSearchInput();
      combobox.updateSelectedOptionIndex("active");
    },
  });
  const current = getProvider(props.value);

  const groups = React.useMemo(() => {
    const ret = new Map<ExporterCategory, typeof PROVIDERS>();
    for (const provider of PROVIDERS) {
      if (
        query &&
        !matches(
          query,
          [
            provider.label,
            provider.description,
            provider.signals.join(" "),
            ...provider.keywords,
          ].join(" "),
        )
      ) {
        continue;
      }
      ret.set(provider.category, [
        ...(ret.get(provider.category) ?? []),
        provider,
      ]);
    }
    return [...ret.entries()];
  }, [query]);

  React.useEffect(() => {
    if (combobox.dropdownOpened) combobox.selectFirstOption();
  }, [query]);

  return (
    <section className="flex min-w-0 flex-col gap-2">
      <div>
        <h3 className="text-body font-semibold text-slate-900">Destination</h3>
        <p className="text-xs font-normal text-slate-500">
          Where the Collector sends the logs and metrics of its pipelines.
        </p>
      </div>
      <Combobox
        store={combobox}
        position="bottom-start"
        offset={6}
        shadow="md"
        radius="lg"
        withinPortal
        transitionProps={{ transition: "pop", duration: 150 }}
        onOptionSubmit={(value) => {
          if (value !== props.value) props.onChange(value as ExporterKind);
          combobox.closeDropdown();
        }}
        styles={{
          dropdown: {
            border: "1px solid var(--color-slate-200)",
            boxShadow: "var(--shadow-overlay)",
            padding: 0,
            overflow: "hidden",
            maxWidth: "calc(100vw - 32px)",
          },
        }}
      >
        <Combobox.Target withAriaAttributes={false}>
          <button
            type="button"
            aria-label={`Destination: ${current?.label ?? "not selected"}. Change the destination`}
            aria-haspopup="listbox"
            aria-expanded={combobox.dropdownOpened}
            onClick={() => combobox.toggleDropdown()}
            className={twMerge(
              "flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-white px-3.5 py-3 text-left outline-none transition-[border-color,box-shadow] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
              combobox.dropdownOpened
                ? "border-slate-400 shadow-raised"
                : "border-slate-200 hover:border-slate-300",
            )}
          >
            <ExporterTile kind={props.value} size="lg" />
            <span className="flex min-w-0 flex-1 flex-col gap-1">
              <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <span className="truncate text-sm font-semibold text-slate-900">
                  {current?.label ?? "Choose a destination"}
                </span>
                {current && <SignalBadges signals={current.signals} />}
              </span>
              <span className="truncate text-xs font-normal text-slate-500">
                {current?.description ??
                  "Pick the backend that receives the telemetry."}
              </span>
            </span>
            <span className="hidden shrink-0 items-center gap-1 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-600 @md:flex">
              Change
              <ChevronsUpDown size={12} strokeWidth={2.4} />
            </span>
            <ChevronsUpDown
              size={14}
              strokeWidth={2.4}
              className="shrink-0 text-slate-500 @md:hidden"
            />
          </button>
        </Combobox.Target>

        <Combobox.Dropdown>
          <div className="border-b border-slate-100 p-2">
            <Combobox.Search
              value={query}
              onChange={(event) => setQuery(event.currentTarget.value)}
              placeholder="Search destinations"
              aria-label="Search destinations"
              leftSection={<Search size={13} className="text-slate-400" />}
              styles={{
                input: {
                  border: "none",
                  background: "var(--color-slate-50)",
                  borderRadius: "8px",
                  minHeight: "34px",
                  margin: 0,
                  width: "100%",
                },
              }}
            />
          </div>
          <Combobox.Options
            aria-label="Destinations"
            className="max-h-[min(420px,60dvh)] overflow-y-auto p-1.5"
          >
            {groups.length === 0 ? (
              <Combobox.Empty className="px-3 py-6 text-xs text-slate-500">
                No destination matches “{query}”. Most observability vendors
                accept OTLP, so try OTLP gRPC or OTLP HTTP.
              </Combobox.Empty>
            ) : (
              groups.map(([category, items]) => (
                <Combobox.Group key={category} label={CATEGORY_LABEL[category]}>
                  {items.map((provider) => {
                    const selected = provider.kind === props.value;
                    return (
                      <Combobox.Option
                        key={provider.kind}
                        value={provider.kind}
                        active={selected}
                        className="rounded-lg data-[combobox-selected]:!bg-slate-100 data-[combobox-selected]:!text-slate-900 data-[combobox-selected]:ring-1 data-[combobox-selected]:ring-slate-200"
                      >
                        <span className="flex min-w-0 items-center gap-3 py-0.5">
                          <ExporterTile kind={provider.kind} size="sm" />
                          <span className="flex min-w-0 flex-1 flex-col">
                            <span className="flex min-w-0 items-center gap-2">
                              <span className="truncate text-body font-semibold text-slate-900">
                                {provider.label}
                              </span>
                              <SignalBadges
                                signals={provider.signals}
                                className="flex-nowrap"
                              />
                            </span>
                            <span className="truncate text-xs font-normal text-slate-500">
                              {provider.description}
                            </span>
                          </span>
                          <Check
                            size={14}
                            strokeWidth={2.75}
                            className={twMerge(
                              "shrink-0 text-slate-700",
                              selected ? "" : "invisible",
                            )}
                          />
                        </span>
                      </Combobox.Option>
                    );
                  })}
                </Combobox.Group>
              ))
            )}
          </Combobox.Options>
        </Combobox.Dropdown>
      </Combobox>
    </section>
  );
};

export default DestinationPicker;
