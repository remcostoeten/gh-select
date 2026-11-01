/**
 * @name currentYear
 * @description The current year for the footer copyright, read when the page is prerendered.
 *
 * @example
 * const year = await currentYear() // 2026
 */
export async function currentYear() {
  return new Date().getFullYear();
}
