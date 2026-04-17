import {
  Accordion,
  Button,
  createTheme,
  Input,
  MultiSelect,
  NumberInput,
  Select,
  Switch,
  Tabs,
  TagsInput,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";

const FONT =
  'Ubuntu, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

const inputClassName =
  "!font-bold !transition-all !duration-500 !rounded-md !focus:shadow-md !focus:border-gray-900 !border-[2px]";

const theme = createTheme({
  fontFamily: FONT,
  fontFamilyMonospace:
    "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  primaryColor: "dark",
  autoContrast: true,
  defaultRadius: "md",

  components: {
    Button: Button.extend({
      defaultProps: {
        variant: "filled",
        className:
          "!font-bold !shadow-md !transition-all !duration-500 !rounded-md",
      },
    }),

    Input: Input.extend({
      styles: {
        input: {
          fontFamily: FONT,
          fontWeight: 600,
        },
      },
    }),

    InputBase: Input.extend({
      styles: {
        input: {
          fontFamily: FONT,
          fontWeight: 600,
        },
      },
    }),

    TextInput: TextInput.extend({
      classNames: {
        label: "!font-bold",
        input: inputClassName,
      },
    }),

    Textarea: Textarea.extend({
      classNames: {
        label: "!font-bold",
        input: inputClassName,
      },
    }),

    NumberInput: NumberInput.extend({
      classNames: {
        label: "!font-bold",
        input: inputClassName,
      },
    }),

    TagsInput: TagsInput.extend({
      classNames: {
        label: "!font-bold",
        input: inputClassName,
      },
    }),

    Accordion: Accordion.extend({
      classNames: {
        label: "!font-bold",
        panel: "!font-bold",
      },
    }),

    Tabs: Tabs.extend({
      classNames: {
        tab: "!font-bold",
        panel: "!font-bold",
      },
    }),

    Switch: Switch.extend({
      classNames: {
        label: "!font-bold",
        input: "!transition-all !duration-500",
      },
    }),

    Select: Select.extend({
      defaultProps: {
        radius: "md",
        comboboxProps: {
          transitionProps: { transition: "pop", duration: 200 },
          shadow: "sm",
          radius: "md",
        },
      },
      classNames: {
        input: inputClassName,
        label: "!font-bold",
        option: "!transition-all !duration-500 !font-bold hover:!bg-zinc-200",
      },
    }),

    MultiSelect: MultiSelect.extend({
      defaultProps: {
        radius: "md",
        comboboxProps: {
          transitionProps: { transition: "pop", duration: 200 },
          shadow: "sm",
          radius: "md",
        },
      },
      classNames: {
        input: inputClassName,
        label: "!font-bold",
        option: "!transition-all !duration-500 !font-bold hover:!bg-zinc-200",
      },
    }),

    Tooltip: Tooltip.extend({
      defaultProps: {
        transitionProps: {
          transition: "fade",
          duration: 350,
        },
        classNames: {
          tooltip: "!shadow-md !font-bold !text-xs !rounded-sm",
        },
      },
    }),
  },
});

export default theme;
