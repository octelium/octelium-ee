import js from "@eslint/js";
import query from "@tanstack/eslint-plugin-query";
import typescript from "@typescript-eslint/eslint-plugin";
import hooks from "eslint-plugin-react-hooks";
import refresh from "eslint-plugin-react-refresh";
import globals from "globals";

const typescriptFiles = ["**/*.{ts,tsx}"];

export default [
  { ignores: ["dist/**", "src/apis/**", "tmp/**"] },
  {
    ...js.configs.recommended,
    files: ["**/*.{js,mjs,cjs}"],
    languageOptions: { globals: globals.node },
  },
  { ...js.configs.recommended, files: typescriptFiles },
  ...typescript.configs["flat/recommended"].map((config) => ({
    ...config,
    files: typescriptFiles,
  })),
  {
    files: typescriptFiles,
    languageOptions: { globals: globals.browser },
    plugins: { "react-hooks": hooks },
    rules: hooks.configs.recommended.rules,
  },
  {
    files: ["src/**/*.{ts,tsx}"],
    ignores: ["src/pages/**/routes.tsx"],
    plugins: { "react-refresh": refresh },
    rules: {
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  },
  ...query.configs["flat/recommended"].map((config) => ({
    ...config,
    files: typescriptFiles,
  })),
  {
    files: ["src/**/*.js"],
    languageOptions: { globals: globals.browser },
  },
  {
    files: ["tests/**/*.mjs"],
    languageOptions: { globals: globals.browser },
  },
  {
    files: ["vite.config.ts"],
    languageOptions: { globals: globals.node },
  },
];
