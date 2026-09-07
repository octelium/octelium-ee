import {
  Accordion,
  Badge,
  Button,
  Checkbox,
  createTheme,
  HoverCard,
  Input,
  Menu,
  MultiSelect,
  NumberInput,
  Pagination,
  Popover,
  Radio,
  rem,
  SegmentedControl,
  Select,
  Switch,
  Tabs,
  TagsInput,
  Textarea,
  TextInput,
  Tooltip,
  type MantineTransition,
} from "@mantine/core";

const FONT =
  'Ubuntu, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

const SIZE_MICRO = "0.6875rem";
const SIZE_META = "0.75rem";
const SIZE_BODY = "0.8125rem";

const labelStyles = {
  label: {
    fontSize: SIZE_META,
    fontWeight: 600,
    fontFamily: FONT,
    color: "#334155",
    marginBottom: "4px",
  },
  description: {
    fontSize: SIZE_META,
    fontWeight: 400,
    fontFamily: FONT,
    lineHeight: 1.45,
    color: "#64748b",
    marginBottom: "6px",
  },
  error: {
    fontSize: SIZE_META,
    fontWeight: 500,
    fontFamily: FONT,
    color: "#dc2626",
  },
};

const inputStyles = {
  input: {
    fontSize: SIZE_BODY,
    fontWeight: 400,
    fontFamily: FONT,
    backgroundColor: "#ffffff",
    border: "1px solid #e2e8f0",
    borderRadius: "8px",
    color: "#0f172a",
    boxShadow: "none",
    minHeight: "38px",
    transition: "border-color 150ms, box-shadow 150ms",
  },
};

const optionStyles = {
  option: {
    fontSize: SIZE_BODY,
    fontWeight: 400,
    fontFamily: FONT,
    borderRadius: "7px",
    minHeight: "34px",
    transition: "background-color 150ms, color 150ms",
  },
};

const comboboxDefaultProps = {
  radius: "md" as const,
  comboboxProps: {
    transitionProps: { transition: "pop" as MantineTransition, duration: 150 },
    shadow: "md",
    radius: "lg" as const,
  },
};

const pillStyles = {
  pill: {
    fontSize: SIZE_MICRO,
    fontWeight: 500,
    fontFamily: FONT,
    backgroundColor: "#f1f5f9",
    color: "#334155",
    border: "1px solid #e2e8f0",
    borderRadius: "6px",
  },
};

const toggleLabelStyles = {
  label: {
    fontSize: SIZE_BODY,
    fontWeight: 500,
    fontFamily: FONT,
    color: "#0f172a",
  },
  description: labelStyles.description,
};

const theme = createTheme({
  fontFamily: FONT,
  fontFamilyMonospace:
    "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  primaryColor: "dark",
  autoContrast: true,
  defaultRadius: "md",
  cursorType: "pointer",

  components: {
    Button: Button.extend({
      defaultProps: { variant: "filled" },
      styles: {
        root: {
          fontFamily: FONT,
          fontWeight: 600,
          fontSize: SIZE_BODY,
          borderRadius: "8px",
          transition:
            "background-color 150ms, border-color 150ms, color 150ms, opacity 150ms",
          boxShadow: "none",
        },
        label: { fontFamily: FONT, fontWeight: 600 },
      },
    }),

    Input: Input.extend({ styles: { ...labelStyles, ...inputStyles } }),

    TextInput: TextInput.extend({ styles: { ...labelStyles, ...inputStyles } }),

    Textarea: Textarea.extend({
      styles: {
        ...labelStyles,
        input: {
          ...inputStyles.input,
          resize: "vertical" as const,
          lineHeight: 1.6,
          paddingTop: "8px",
          paddingBottom: "8px",
        },
      },
    }),

    NumberInput: NumberInput.extend({
      styles: { ...labelStyles, ...inputStyles },
    }),

    TagsInput: TagsInput.extend({
      styles: { ...labelStyles, ...inputStyles, ...pillStyles },
    }),

    MultiSelect: MultiSelect.extend({
      defaultProps: comboboxDefaultProps,
      styles: {
        ...labelStyles,
        ...inputStyles,
        ...pillStyles,
        ...optionStyles,
      },
    }),

    Select: Select.extend({
      defaultProps: comboboxDefaultProps,
      styles: { ...labelStyles, ...inputStyles, ...optionStyles },
    }),

    Switch: Switch.extend({
      styles: {
        ...toggleLabelStyles,
        track: { transition: "background-color 150ms", cursor: "pointer" },
        thumb: { transition: "inset-inline-start 150ms" },
      },
    }),

    Checkbox: Checkbox.extend({
      styles: {
        ...toggleLabelStyles,
        input: {
          cursor: "pointer",
          borderColor: "#cbd5e1",
          transition: "background-color 150ms, border-color 150ms",
        },
      },
    }),

    Radio: Radio.extend({
      styles: {
        ...toggleLabelStyles,
        radio: {
          cursor: "pointer",
          borderColor: "#cbd5e1",
          transition: "background-color 150ms, border-color 150ms",
        },
      },
    }),

    SegmentedControl: SegmentedControl.extend({
      styles: {
        root: {
          backgroundColor: "#f1f5f9",
          border: "1px solid #e2e8f0",
          borderRadius: "10px",
          padding: "3px",
        },
        label: {
          fontSize: SIZE_BODY,
          fontWeight: 600,
          color: "#64748b",
          transition: "color 150ms",
        },
        indicator: {
          backgroundColor: "#ffffff",
          borderRadius: "7px",
          boxShadow: "0 1px 2px rgba(15,23,42,0.12)",
          transition: "transform 150ms ease, width 150ms ease",
        },
      },
    }),

    Pagination: Pagination.extend({
      styles: {
        control: {
          fontFamily: FONT,
          fontWeight: 600,
          fontSize: SIZE_BODY,
          border: "1px solid #e2e8f0",
          backgroundColor: "#ffffff",
          color: "#475569",
          boxShadow: "none",
          transition: "background-color 150ms, border-color 150ms, color 150ms",
        },
      },
    }),

    Menu: Menu.extend({
      defaultProps: {
        shadow: "md",
        radius: "lg",
        transitionProps: {
          transition: "pop" as MantineTransition,
          duration: 150,
        },
      },
      styles: {
        dropdown: {
          border: "1px solid #e2e8f0",
          boxShadow: "0 16px 40px rgba(15,23,42,0.14)",
          padding: "6px",
        },
        item: {
          borderRadius: "7px",
          fontFamily: FONT,
          fontSize: SIZE_BODY,
          fontWeight: 500,
          minHeight: "34px",
          transition: "background-color 150ms, color 150ms",
        },
        label: {
          color: "#64748b",
          fontFamily: FONT,
          fontSize: SIZE_MICRO,
          fontWeight: 600,
          letterSpacing: "0.05em",
          textTransform: "uppercase",
        },
        divider: { borderColor: "#e2e8f0" },
      },
    }),

    Tabs: Tabs.extend({
      styles: {
        tab: {
          fontSize: SIZE_BODY,
          fontWeight: 600,
          fontFamily: FONT,
          transition: "color 150ms, border-color 150ms",
        },
        panel: { fontFamily: FONT, marginTop: rem(16) },
      },
    }),

    Accordion: Accordion.extend({
      styles: {
        label: {
          fontSize: SIZE_BODY,
          fontWeight: 600,
          fontFamily: FONT,
          color: "#0f172a",
        },
        panel: { fontFamily: FONT },
        control: { transition: "background-color 150ms" },
      },
    }),

    Badge: Badge.extend({
      styles: {
        root: {
          fontFamily: FONT,
          fontWeight: 600,
          fontSize: SIZE_MICRO,
          letterSpacing: "0.02em",
        },
        label: { fontFamily: FONT, fontWeight: 600 },
      },
    }),

    Tooltip: Tooltip.extend({
      defaultProps: {
        transitionProps: {
          transition: "fade" as MantineTransition,
          duration: 150,
        },
      },
      styles: {
        tooltip: {
          fontFamily: FONT,
          fontWeight: 400,
          fontSize: SIZE_META,
          backgroundColor: "#0f172a",
          color: "#f8fafc",
          border: "1px solid #1e293b",
          borderRadius: "8px",
          boxShadow: "0 4px 12px rgba(15,23,42,0.15)",
          padding: "5px 10px",
        },
      },
    }),

    HoverCard: HoverCard.extend({
      defaultProps: {
        shadow: "md",
        withArrow: true,
        openDelay: 200,
        closeDelay: 400,
        transitionProps: { transition: "pop" as MantineTransition },
      },
      styles: {
        dropdown: {
          border: "1px solid #e2e8f0",
          borderRadius: "12px",
          boxShadow: "0 16px 40px rgba(15,23,42,0.14)",
          fontFamily: FONT,
        },
      },
    }),

    Popover: Popover.extend({
      defaultProps: {
        shadow: "xl",
        withArrow: true,
        transitionProps: {
          transition: "pop" as MantineTransition,
          duration: 150,
        },
      },
      styles: {
        dropdown: {
          border: "1px solid #e2e8f0",
          borderRadius: "12px",
          boxShadow: "0 16px 40px rgba(15,23,42,0.14)",
          fontFamily: FONT,
          overflow: "visible",
        },
        arrow: { border: "1px solid #e2e8f0" },
      },
    }),
  },
});

export default theme;
