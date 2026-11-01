import { defineConfig } from "vite-plus";

export default defineConfig({
  fmt: {
    semi: true,
    singleQuote: false,
    ignorePatterns: [".next/**", ".source/**", "bun.lock", "next-env.d.ts"],
  },
  lint: {
    plugins: ["typescript", "unicorn", "oxc", "import", "react", "jsx-a11y", "nextjs"],
    ignorePatterns: [".next/**", ".source/**"],
    rules: {
      "func-style": ["error", "declaration", { allowArrowFunctions: false }],
      "prefer-arrow-callback": "error",
    },
  },
});
