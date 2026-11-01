/**
 * @name noop
 * @description Does nothing. Marks a deliberately ignored outcome, such as a
 * storage write that may fail in private browsing.
 *
 * @example
 * try { localStorage.setItem(key, value) } catch { noop() }
 */
export function noop() {}
