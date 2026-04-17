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
    letterSpacing: "0.05em",
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
    borderRadius: "6px",
    color: "#1e293b",
    boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
    transition: "border-color 150ms, box-shadow 150ms",
    "&:focus": {
      borderColor: "#94a3b8",
      boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
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
    borderRadius: "4px",
    transition: "background-color 100ms",
  },
};

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
  primaryColor: "dark",
  autoContrast: true,
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
          transition: "background-color 150ms, box-shadow 150ms",
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
          minHeight: "36px",
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
          color: "#334155",
        },
        description: labelStyles.description,
        input: {
          cursor: "pointer",
          borderColor: "#e2e8f0",
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
          color: "#334155",
        },
        description: labelStyles.description,
        radio: {
          cursor: "pointer",
          borderColor: "#e2e8f0",
          transition: "background-color 150ms, border-color 150ms",
        },
      },
    }),

    SegmentedControl: SegmentedControl.extend({
      styles: {
        root: {
          backgroundColor: "#f1f5f9",
          border: "1px solid #e2e8f0",
          borderRadius: "6px",
          padding: "2px",
        },
        label: {
          fontSize: "0.72rem",
          fontWeight: 700,
          fontFamily: FONT,
          color: "#64748b",
          transition: "color 150ms",
        },
        indicator: {
          backgroundColor: "#0f172a",
          borderRadius: "4px",
          boxShadow: "none",
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
          boxShadow: "0 1px 3px rgba(15,23,42,0.07)",
          transition: "background-color 150ms, border-color 150ms",
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
          color: "#1e293b",
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
          backgroundColor: "#1e293b",
          color: "#f8fafc",
          border: "1px solid #334155",
          borderRadius: "6px",
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
          borderRadius: "10px",
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
          borderRadius: "10px",
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
