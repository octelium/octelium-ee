import * as Yaml from "js-yaml";

const schema = Yaml.CORE_SCHEMA.withTags(
  Yaml.mergeTag,
  Yaml.timestampTag,
  Yaml.binaryTag,
  Yaml.omapTag,
  Yaml.pairsTag,
  Yaml.legacyMapTag,
  {
    ...Yaml.intCoreTag,
    resolve: (source: string, _isExplicit: boolean, tagName: string) =>
      Yaml.intCoreTag.resolve(source, true, tagName),
  },
  {
    ...Yaml.floatCoreTag,
    resolve: (source: string, isExplicit: boolean, tagName: string) =>
      /^[-+]\.[0-9]/.test(source)
        ? Yaml.NOT_RESOLVED
        : Yaml.floatCoreTag.resolve(source, isExplicit, tagName),
  },
  {
    ...Yaml.legacyMapTag,
    tagName: Yaml.setTag.tagName,
    addPair: (
      container: Record<string, unknown>,
      key: unknown,
      value: unknown,
    ) =>
      value === null
        ? Yaml.legacyMapTag.addPair(container, key, value)
        : "Set values must be null",
    identify: () => false,
  },
);

export const loadYAML = (arg: string): unknown => {
  if (arg.split(/[\r\n]/).every((line) => /^\s*(?:#.*)?$/.test(line))) {
    return undefined;
  }
  return Yaml.load(arg, { schema });
};
