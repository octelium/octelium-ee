const flags = import.meta.glob<string>(
  "/node_modules/flag-icons/flags/4x3/*.svg",
  { eager: true, query: "?url&no-inline", import: "default" },
);

export const CountryFlag = (props: { code?: string; className?: string }) => {
  const code = props.code?.trim().toLowerCase();
  if (!code || code.length !== 2) return null;
  const src = flags[`/node_modules/flag-icons/flags/4x3/${code}.svg`];
  if (!src) return null;

  return (
    <img
      src={src}
      alt=""
      aria-hidden="true"
      loading="lazy"
      className={props.className ?? "shrink-0 rounded-[2px]"}
      style={props.className ? undefined : { width: "1.4rem", height: "1.05rem" }}
    />
  );
};

export default CountryFlag;
