import {camelToSentence} from '@/util/string';
import {toValue} from 'vue';

/**
 * Escapes a single field for RFC 4180 CSV output. Wraps the value in double
 * quotes when it contains a comma, double-quote, CR or LF, doubling any
 * embedded double-quotes. Nullish values become an empty field.
 *
 * Note: no spreadsheet formula-injection guard (prefixing =/+/-/@) is applied,
 * as that would silently alter legitimate negative numeric values in exports.
 *
 * @param {*} field
 * @return {string}
 */
function escapeCSVField(field) {
  const str = field === null || field === undefined ? '' : String(field);
  if (/[",\r\n]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

/**
 * Generates a function to download records as a CSV file based on specified properties.
 *
 * @template T
 * @param {{
 *   acronyms: Object<string, {label: string, unit: string}>,
 *   docType: string,
 *   flattenRecords: (records: Array<Object>) => Array<Object>,
 *   records: function(): Promise<T[]>,
 *   deviceName: MaybeRefOrGetter<string>
 * }} params An object containing all configurations:
 *          - `acronyms`: Maps acronyms to their full forms for clarity in the CSV content.
 *          - `docType`: The type of document to be generated, affecting the CSV structure.
 *          - `flattenRecords`: A function that flattens the array of record objects for CSV conversion.
 *          - `records`: An async function returning an array of record objects for the CSV.
 *          - `deviceName`: The deviceName from where the records are fetched, tailoring CSV content.
 *
 * Returned an object containing the `downloadCSV` method,
 * which generates and then downloads the CSV file asynchronously.
 * @return {void}
 */
export const csvDownload = async (params) => {
  /**
   * Capitalize the first letter of a string
   *
   * @param {string} str
   * @return {string}
   */
  const capitalize = (str) => str.charAt(0).toUpperCase() + str.slice(1);

  /**
   * Create a mapping of headers for the CSV file
   *
   * @param {Array} flattenedRecords
   * @return {Object}
   */
  const createHeaderMap = (flattenedRecords) => {
    const headerMap = {};
    Object.keys(flattenedRecords[0]).forEach(key => {
      // Fallback to camelToSentence if label is not defined
      const label = params.acronyms[key]?.label || camelToSentence(key);
      // Add unit if it exists
      const unit = params.acronyms[key]?.unit ? `(${params.acronyms[key].unit})` : '';
      headerMap[key] = `${capitalize(label)}${unit !== '' ? ' ' + unit : ''}`;
    });
    return headerMap;
  };

  /**
   * Structure records into CSV format
   *
   * @param {Array} records
   * @return {string}
   */
  const processRecordsForCSV = (records) => {
    // Check if there are records to process
    if (!records || !records.length) return '';

    // Flatten the records
    const flattenedRecords = params.flattenRecords(records);

    // Create a mapping for headers
    const headerMap = createHeaderMap(flattenedRecords);

    // Generate the header row
    const header = Object.values(headerMap);

    // Generate the data rows
    const rows = flattenedRecords.map(record =>
      Object.keys(headerMap).map(key => escapeCSVField(record[key])).join(',')
    );

    // Return the CSV string
    return [header.map(escapeCSVField).join(',')].concat(rows).join('\n');
  };


  /**
   * Download the records as a CSV file
   *
   * @return {Promise<void>}
   */
  const download = async () => {
    // Check if there's a selected deviceName, if not, log a warning and return
    if (!toValue(params.deviceName)) {
      console.warn('No deviceName selected for downloading CSV');
      return;
    }

    // Reassign records to the available records
    const availableRecords = await params.records();

    // Check if there are records to process, if not, log a warning and return
    if (!availableRecords || availableRecords.length === 0) {
      console.warn('No records found for deviceName: ', toValue(params.deviceName));
      return;
    }

    // Process records to CSV format
    const csvString = processRecordsForCSV(availableRecords);

    // Create a Blob from the CSV String
    const blob = new Blob([csvString], {type: 'text/csv;charset=utf-8;'});
    // Create a link element to trigger the download
    const link = document.createElement('a');
    // Create a URL for the Blob
    const url = URL.createObjectURL(blob);

    // Trigger the download
    const date = new Date();
    // Convert the date to a string
    const dateString = `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
    // Set the link attributes
    link.href = url;
    // Set the download attribute
    link.setAttribute('download', `${params.docType} - ${toValue(params.deviceName)} - ${dateString}.csv`);
    // Append the link to the body
    document.body.appendChild(link);
    // Trigger the click event
    link.click();
    // Remove the link from the body
    document.body.removeChild(link);
    // Revoke the URL
    URL.revokeObjectURL(url);
  };

  await download();
};

/**
 * Causes the browser to download a CSV file containing the given row data.
 *
 * @param {string} filename
 * @param {string[][]} data
 */
export function downloadCSVRows(filename, data) {
  const csvContent = data.map(e => e.map(escapeCSVField).join(',')).join('\n') + '\n';
  download(filename, csvContent, 'text/csv;charset=utf-8;');
}

/**
 * @param {string} filename
 * @param {string} text
 * @param {string} mime
 */
function download(filename, text, mime) {
  const element = document.createElement('a');
  element.setAttribute('href', 'data:' + mime + ',' + encodeURIComponent(text));
  element.setAttribute('download', filename);

  element.style.display = 'none';
  document.body.appendChild(element);

  element.click();

  document.body.removeChild(element);
}
