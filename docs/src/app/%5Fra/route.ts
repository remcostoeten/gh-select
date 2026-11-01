import { createProxy } from "@spoar/sdk/proxy";

export const POST = createProxy({
  secret: process.env.SPOAR_SECRET,
  endpoint: process.env.SPOAR_ENDPOINT ?? "https://api.analytics.remcostoeten.nl",
});
