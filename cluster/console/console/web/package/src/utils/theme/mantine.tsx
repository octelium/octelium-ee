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

const labelStyles = {
  label: {
    fontSize: "0.72rem",
    fontWeight: 700,
    fontFamily: FONT,
    textTransform: "uppercase" as const,
    letterSpacing: "0.04em",
    color: "#475569",
    marginBottom: "4px",
  },
  description: {
    fontSize: "0.7rem",
    fontWeight: 600,
    fontFamily: FONT,
    color: "#94a3b8",
    marginBottom: "4px",
  },
  error: {
    fontSize: "0.7rem",
    fontWeight: 600,
    fontFamily: FONT,
  },
};

const inputStyles = {
  input: {
    fontSize: "0.82rem",
    fontWeight: 600,
    fontFamily: FONT,
    backgroundColor: "#ffffff",
    border: "1px solid #e2e8f0",
    borderRadius: "8px",
    color: "#1e293b",
    boxShadow: "none",
    minHeight: "38px",
    transition: "border-color 500ms, background-color 500ms",
    "&:focus, &[data-focus]": {
      borderColor: "#64748b",
      boxShadow: "none",
      outline: "none",
    },
    "&[data-error]": {
      borderColor: "#ef4444",
    },
    "&:disabled": {
      backgroundColor: "#f8fafc",
      color: "#94a3b8",
      borderColor: "#e2e8f0",
      cursor: "not-allowed",
    },
    "&::placeholder": {
      color: "#94a3b8",
      fontWeight: 600,
    },
  },
};

const optionStyles = {
  option: {
    fontSize: "0.78rem",
    fontWeight: 600,
    fontFamily: FONT,
    borderRadius: "7px",
    minHeight: "34px",
    transition: "background-color 300ms, color 300ms",
    "&[data-combobox-selected]": {
      backgroundColor: "#0f172a",
      color: "#ffffff",
    },
  },
};

const comboboxDefaultProps = {
  radius: "md" as const,
  comboboxProps: {
    transitionProps: { transition: "pop" as MantineTransition, duration: 200 },
    shadow: "md",
    radius: "lg" as const,
  },
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
      defaultProps: {
        variant: "filled",
      },
      styles: {
        root: {
          fontFamily: FONT,
          fontWeight: 700,
          fontSize: "0.78rem",
          borderRadius: "8px",
          transition:
            "background-color 500ms, border-color 500ms, color 500ms, opacity 500ms",
          boxShadow: "none",
          "&:focus-visible": {
            outline: "2px solid #94a3b8",
            outlineOffset: "2px",
          },
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
          minHeight: "38px",
        },
        pill: {
          fontSize: "0.7rem",
          fontWeight: 700,
          fontFamily: FONT,
          backgroundColor: "#f1f5f9",
          color: "#334155",
          border: "1px solid #e2e8f0",
        },
      },
    }),

    MultiSelect: MultiSelect.extend({
      defaultProps: comboboxDefaultProps,
      styles: {
        ...labelStyles,
        input: {
          ...inputStyles.input,
          minHeight: "38px",
        },
        pill: {
          fontSize: "0.7rem",
          fontWeight: 700,
          fontFamily: FONT,
          backgroundColor: "#f1f5f9",
          color: "#334155",
          border: "1px solid #e2e8f0",
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
          color: "#334155",
        },
        description: labelStyles.description,
        track: {
          transition: "background-color 300ms, border-color 300ms",
          cursor: "pointer",
        },
        thumb: {
          transition: "left 300ms",
        },
      },
    }),

    Checkbox: Checkbox.extend({
      styles: {
        label: {
          fontSize: "0.78rem",
          fontWeight: 600,
          fontFamily: FONT,
          color: "#334155",
        },
        description: labelStyles.description,
        input: {
          cursor: "pointer",
          borderColor: "#e2e8f0",
          transition: "background-color 300ms, border-color 300ms",
        },
      },
    }),

    Radio: Radio.extend({
      styles: {
        label: {
          fontSize: "0.78rem",
          fontWeight: 600,
          fontFamily: FONT,
          color: "#334155",
        },
        description: labelStyles.description,
        radio: {
          cursor: "pointer",
          borderColor: "#e2e8f0",
          transition: "background-color 300ms, border-color 300ms",
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
          fontSize: "0.82rem",
          fontWeight: 700,
          color: "#64748b",
          transition: "color 500ms",
          "&[data-active]": {
            color: "#0f172a",
          },
        },
        indicator: {
          backgroundColor: "#ffffff",
          borderRadius: "7px",
          boxShadow: "0 1px 3px rgba(15,23,42,0.16)",
          transition: "transform 300ms ease, width 300ms ease",
        },
      },
    }),

    Pagination: Pagination.extend({
      styles: {
        control: {
          fontFamily: FONT,
          fontWeight: 700,
          fontSize: "0.78rem",
          border: "1px solid #e2e8f0",
          backgroundColor: "#ffffff",
          color: "#475569",
          boxShadow: "none",
          transition:
            "background-color 500ms, border-color 500ms, color 500ms",
          "&:hover": {
            backgroundColor: "#f8fafc",
            borderColor: "#cbd5e1",
          },
          "&[data-active]": {
            backgroundColor: "#0f172a",
            borderColor: "#0f172a",
            color: "#ffffff",
          },
        },
      },
    }),

    Menu: Menu.extend({
      defaultProps: {
        shadow: "md",
        radius: "lg",
        transitionProps: {
          transition: "pop" as MantineTransition,
          duration: 200,
        },
      },
      styles: {
        dropdown: {
          border: "1px solid #e2e8f0",
          boxShadow: "0 10px 28px rgba(15,23,42,0.12)",
          padding: "6px",
        },
        item: {
          borderRadius: "7px",
          fontFamily: FONT,
          fontSize: "0.76rem",
          fontWeight: 700,
          minHeight: "34px",
          transition: "background-color 300ms, color 300ms",
        },
        label: {
          color: "#94a3b8",
          fontFamily: FONT,
          fontSize: "0.64rem",
          fontWeight: 700,
          letterSpacing: "0.05em",
          textTransform: "uppercase",
        },
        divider: { borderColor: "#e2e8f0" },
      },
    }),

    Tabs: Tabs.extend({
      styles: {
        tab: {
          fontSize: "0.78rem",
          fontWeight: 700,
          fontFamily: FONT,
          transition: "color 500ms, border-color 500ms",
        },
        panel: {
          fontFamily: FONT,
          fontWeight: 600,
          marginTop: rem(16),
        },
      },
    }),

    Accordion: Accordion.extend({
      styles: {
        label: {
          fontSize: "0.82rem",
          fontWeight: 700,
          fontFamily: FONT,
          color: "#1e293b",
        },
        panel: {
          fontFamily: FONT,
          fontWeight: 600,
        },
        control: {
          transition: "background-color 500ms",
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
          backgroundColor: "#1e293b",
          color: "#f8fafc",
          border: "1px solid #334155",
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
          boxShadow: "0 8px 24px rgba(15,23,42,0.10)",
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
          border: "1px solid #e2e8f0",
          borderRadius: "12px",
          boxShadow: "0 8px 32px rgba(15,23,42,0.12)",
          fontFamily: FONT,
          overflow: "visible",
        },
        arrow: {
          border: "1px solid #e2e8f0",
        },
      },
    }),
  },
});

export default theme;
