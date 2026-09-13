import {
  Accordion,
  Badge,
  Button,
  Checkbox,
  createTheme,
  HoverCard,
  Input,
  MultiSelect,
  NumberInput,
  Pagination,
  Popover,
  Radio,
  SegmentedControl,
  Select,
  Switch,
  Tabs,
  TagsInput,
  Textarea,
  TextInput,
  Tooltip,
  virtualColor,
  type MantineColorsTuple,
  type MantineTransition,
} from "@mantine/core";

const FONT =
  'Ubuntu, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

const labelStyles = {
  label: {
    fontSize: "0.78rem",
    fontWeight: 700,
    fontFamily: FONT,
    textTransform: "uppercase" as const,
    letterSpacing: "0.05em",
    color: "var(--color-slate-600)",
    marginBottom: "4px",
  },
  description: {
    fontSize: "0.75rem",
    fontWeight: 600,
    fontFamily: FONT,
    color: "var(--color-slate-400)",
    marginBottom: "4px",
  },
  error: {
    fontSize: "0.75rem",
    fontWeight: 600,
    fontFamily: FONT,
  },
};

const inputStyles = {
  input: {
    fontSize: "0.82rem",
    fontWeight: 600,
    fontFamily: FONT,
    backgroundColor: "var(--color-white)",
    border: "1px solid var(--color-slate-200)",
    borderRadius: "6px",
    color: "var(--color-slate-800)",
    boxShadow: "var(--shadow-card)",
    transition: "border-color 500ms, box-shadow 500ms",
    "&:focus, &[data-focus]": {
      borderColor: "var(--color-slate-900)",
      boxShadow:
        "0 0 0 2px color-mix(in oklab, var(--color-slate-900) 12%, transparent)",
      outline: "none",
    },
    "&:disabled": {
      backgroundColor: "var(--color-slate-50)",
      color: "var(--color-slate-400)",
      borderColor: "var(--color-slate-200)",
      cursor: "not-allowed",
    },
    "&::placeholder": {
      color: "var(--color-slate-400)",
      fontWeight: 600,
    },
  },
};

const optionStyles = {
  option: {
    fontSize: "0.78rem",
    fontWeight: 600,
    fontFamily: FONT,
    borderRadius: "4px",
    transition: "background-color 100ms",
  },
};

const INK_LIGHT: MantineColorsTuple = [
  "#c9c9c9",
  "#b8b8b8",
  "#828282",
  "#696969",
  "#424242",
  "#3b3b3b",
  "#2e2e2e",
  "#242424",
  "#1f1f1f",
  "#141414",
];

const INK_DARK: MantineColorsTuple = [
  "#99a6b8",
  "#a6b2c3",
  "#b4bfce",
  "#c2cbd9",
  "#cdd5e0",
  "#d8dfe8",
  "#dfe5ed",
  "#e5eaf1",
  "#eaeff5",
  "#d4dbe6",
];

const comboboxDefaultProps = {
  radius: "md" as const,
  comboboxProps: {
    transitionProps: { transition: "pop" as MantineTransition, duration: 180 },
    shadow: "sm",
    radius: "md" as const,
  },
};

const theme = createTheme({
  fontFamily: FONT,
  fontFamilyMonospace:
    "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  primaryColor: "ink",
  autoContrast: true,

  colors: {
    inkLight: INK_LIGHT,
    inkDark: INK_DARK,
    ink: virtualColor({ name: "ink", light: "inkLight", dark: "inkDark" }),
  },
  defaultRadius: "md",

  components: {
    Button: Button.extend({
      defaultProps: {
        variant: "filled",
      },
      styles: {
        root: {
          fontFamily: FONT,
          fontWeight: 700,
          fontSize: "0.78rem",
          borderRadius: "6px",
          transition: "background-color 500ms, box-shadow 500ms",
          boxShadow: "var(--shadow-card)",
        },
        label: {
          fontFamily: FONT,
          fontWeight: 700,
        },
      },
    }),

    Input: Input.extend({
      styles: {
        ...labelStyles,
        ...inputStyles,
      },
    }),

    TextInput: TextInput.extend({
      styles: {
        ...labelStyles,
        ...inputStyles,
      },
    }),

    Textarea: Textarea.extend({
      styles: {
        ...labelStyles,
        input: {
          ...inputStyles.input,
          resize: "vertical" as const,
          lineHeight: "1.6",
        },
      },
    }),

    NumberInput: NumberInput.extend({
      styles: {
        ...labelStyles,
        ...inputStyles,
      },
    }),

    TagsInput: TagsInput.extend({
      styles: {
        ...labelStyles,
        input: {
          ...inputStyles.input,
          minHeight: "36px",
        },
        pill: {
          fontSize: "0.7rem",
          fontWeight: 700,
          fontFamily: FONT,
          backgroundColor: "var(--color-slate-100)",
          color: "var(--color-slate-700)",
          border: "1px solid var(--color-slate-200)",
        },
      },
    }),

    MultiSelect: MultiSelect.extend({
      defaultProps: comboboxDefaultProps,
      styles: {
        ...labelStyles,
        input: {
          ...inputStyles.input,
          minHeight: "36px",
        },
        pill: {
          fontSize: "0.7rem",
          fontWeight: 700,
          fontFamily: FONT,
          backgroundColor: "var(--color-slate-100)",
          color: "var(--color-slate-700)",
          border: "1px solid var(--color-slate-200)",
        },
        ...optionStyles,
      },
    }),

    Select: Select.extend({
      defaultProps: comboboxDefaultProps,
      styles: {
        ...labelStyles,
        ...inputStyles,
        ...optionStyles,
      },
    }),

    Switch: Switch.extend({
      styles: {
        label: {
          fontSize: "0.78rem",
          fontWeight: 600,
          fontFamily: FONT,
          color: "var(--color-slate-700)",
        },
        description: labelStyles.description,
        track: {
          transition: "background-color 150ms",
          cursor: "pointer",
        },
        thumb: {
          transition: "left 150ms",
        },
      },
    }),

    Checkbox: Checkbox.extend({
      styles: {
        label: {
          fontSize: "0.78rem",
          fontWeight: 600,
          fontFamily: FONT,
          color: "var(--color-slate-700)",
        },
        description: labelStyles.description,
        input: {
          cursor: "pointer",
          borderColor: "var(--color-slate-200)",
          transition: "background-color 150ms, border-color 150ms",
        },
      },
    }),

    Radio: Radio.extend({
      styles: {
        label: {
          fontSize: "0.78rem",
          fontWeight: 600,
          fontFamily: FONT,
          color: "var(--color-slate-700)",
        },
        description: labelStyles.description,
        radio: {
          cursor: "pointer",
          borderColor: "var(--color-slate-200)",
          transition: "background-color 150ms, border-color 150ms",
        },
      },
    }),

    SegmentedControl: SegmentedControl.extend({
      styles: {
        root: {
          backgroundColor: "var(--color-slate-100)",
          border: "1px solid var(--color-slate-200)",
        },
        label: {
          fontSize: "0.82rem",
          fontWeight: 700,
          color: "var(--color-slate-500)",
          transition: "color 150ms",
          "&[data-active]": {
            color: "var(--color-slate-900)",
          },
        },
        indicator: {
          backgroundColor: "var(--color-white)",
          borderRadius: "7px",
          boxShadow: "var(--shadow-card)",
        },
      },
    }),

    Pagination: Pagination.extend({
      styles: {
        control: {
          fontFamily: FONT,
          fontWeight: 700,
          fontSize: "0.78rem",
          border: "1px solid var(--color-slate-200)",
          backgroundColor: "var(--color-white)",
          color: "var(--color-slate-600)",
          boxShadow: "var(--shadow-card)",
          transition: "background-color 150ms, border-color 150ms",
          "&:hover": {
            backgroundColor: "var(--color-slate-50)",
            borderColor: "var(--color-slate-300)",
          },
          "&[data-active]": {
            backgroundColor: "var(--color-slate-900)",
            borderColor: "var(--color-slate-900)",
            color: "var(--color-white)",
          },
        },
      },
    }),

    Tabs: Tabs.extend({
      styles: {
        tab: {
          fontSize: "0.78rem",
          fontWeight: 700,
          fontFamily: FONT,
          transition: "color 150ms, border-color 150ms",
        },
        panel: {
          fontFamily: FONT,
          fontWeight: 600,
        },
      },
    }),

    Accordion: Accordion.extend({
      styles: {
        label: {
          fontSize: "0.82rem",
          fontWeight: 700,
          fontFamily: FONT,
          color: "var(--color-slate-800)",
        },
        panel: {
          fontFamily: FONT,
          fontWeight: 600,
        },
        control: {
          transition: "background-color 150ms",
        },
      },
    }),

    Badge: Badge.extend({
      styles: {
        root: {
          fontFamily: FONT,
          fontWeight: 700,
          fontSize: "0.65rem",
          letterSpacing: "0.04em",
        },
        label: {
          fontFamily: FONT,
          fontWeight: 700,
        },
      },
    }),

    Tooltip: Tooltip.extend({
      defaultProps: {
        transitionProps: {
          transition: "fade" as MantineTransition,
          duration: 200,
        },
      },
      styles: {
        tooltip: {
          fontFamily: FONT,
          fontWeight: 600,
          fontSize: "0.75rem",
          backgroundColor: "var(--color-slate-800)",
          color: "var(--color-slate-50)",
          border: "1px solid var(--color-slate-700)",
          borderRadius: "6px",
          boxShadow: "var(--shadow-raised)",
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
          border: "1px solid var(--color-slate-200)",
          borderRadius: "10px",
          boxShadow: "var(--shadow-overlay)",
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
          duration: 180,
        },
      },
      styles: {
        dropdown: {
          border: "1px solid var(--color-slate-200)",
          borderRadius: "10px",
          boxShadow: "var(--shadow-overlay)",
          fontFamily: FONT,
          overflow: "visible",
        },
        arrow: {
          border: "1px solid var(--color-slate-200)",
        },
      },
    }),
  },
});

export default theme;
