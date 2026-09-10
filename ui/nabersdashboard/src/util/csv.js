/**
 * CSV assembly and download, shared by the dashboard's exports.
 *
 * Every export on this dashboard is read by someone outside it — an accredited
 * assessor, an engineer chasing a board — so they all carry a provenance preamble
 * and all quote their cells the same way. That was duplicated verbatim in each
 * component until there were three of them.
 *
 * @module util/csv
 */

/**
 * Characters that make a spreadsheet treat a cell as a formula rather than text.
 *
 * Excel and Sheets both do this on open, and these exports are opened in one by
 * definition — the whole point of them is that an assessor takes them away. The
 * cells are not all ours either: meter titles and refs come from device metadata,
 * so an installer typing `=EM/118` into an appearance title is enough.
 */
const FORMULA_LEAD = /^[=+\-@\t\r]/;

/**
 * A plain number, which must be left alone even though it can start with `-`.
 *
 * Prefixing `-12.5` would land it in the spreadsheet as text, and these exports
 * exist to be summed. Anything that only looks numeric — `-12-5`, `-1+cmd` — is
 * not matched here and so is still defused.
 */
const PLAIN_NUMBER = /^-?\d+(\.\d+)?([eE][+-]?\d+)?$/;

/**
 * Quote a value for CSV where it needs it, and defuse it where it would
 * otherwise be read as a formula.
 *
 * Meter titles contain commas, refs contain slashes and the disclosure preambles
 * are whole sentences, so an unquoted cell silently shifts every column after it.
 *
 * Formula-leading cells are prefixed with a single quote, which is what
 * spreadsheets read as "this is text". Quoting alone does not help: `"=A1"` is
 * still evaluated.
 *
 * @param {*} v
 * @return {string}
 */
export function csvCell(v) {
  let s = v === null || v === undefined ? '' : String(v);
  if (FORMULA_LEAD.test(s) && !PLAIN_NUMBER.test(s)) s = `'${s}`;
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

/**
 * Join rows of cells into a CSV document.
 *
 * @param {Array<Array<*>>} rows an empty row renders as a blank separator line
 * @return {string}
 */
export function toCsv(rows) {
  return rows.map(row => row.map(csvCell).join(',')).join('\n');
}

/**
 * Assemble and download a CSV.
 *
 * @param {Array<Array<*>>} rows
 * @param {string} filename
 */
export function downloadCsv(rows, filename) {
  const blob = new Blob([toCsv(rows)], {type: 'text/csv'});
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  // Deferred. Revoking in the same task races the download the click just
  // queued, and some browsers then save an empty file or nothing at all. A task
  // boundary is enough for the fetch of the blob to have started.
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
