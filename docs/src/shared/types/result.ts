export type ActionResult<Data = undefined> =
  | { ok: true; data: Data }
  | { ok: false; error: string };
