<h1 align="center">gh-select docs</h1>

Documentation site for [gh-select](../README.md), built on Next.js and Fumadocs. Pages are MDX files in `content/docs`, and the site name, theme and footer are set in `src/shared/config/site.ts`.

## Development

Requires [Bun](https://bun.sh).

```bash
bun install        # install dependencies and generate .source
bun dev            # start the dev server on localhost:3000
bun run check      # format and lint through vp
bun run typecheck  # regenerate .source and route types, then tsc
bun test           # run the tests
bun run build      # sync the changelog, check internal links, then build for production
```

Set `NEXT_PUBLIC_SITE_URL` to the production URL before `bun run build`. The changelog page is generated from `../CHANGELOG.md` and is not edited by hand. When a command, option, environment variable or key changes, update its page under `content/docs/reference` in the same change.
