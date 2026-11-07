import type { Terminal } from "@xterm/xterm";

export const terminalFontFamily =
  '"JetBrains Mono", ui-monospace, Menlo, Consolas, monospace';

export const loadTerminalFont = () =>
  Promise.allSettled([
    document.fonts.load('400 1em "JetBrains Mono"'),
    document.fonts.load('700 1em "JetBrains Mono"'),
  ]);

export const refreshTerminalFont = (terminal: Terminal) => {
  terminal.options.fontFamily = "monospace";
  terminal.options.fontFamily = terminalFontFamily;
};
