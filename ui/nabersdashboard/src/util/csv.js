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
 * Quote a value for CSV where it needs it.
 *
 * Meter titles contain commas, refs contain slashes and the disclosure preambles
 * are whole sentences, so an unquoted cell silently shifts every column after it.
 *
 * @param {*} v
 * @return {string}
 */
export function csvCell(v) {
  const s = v === null || v === undefined ? '' : String(v);
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
  URL.revokeObjectURL(url);
}
