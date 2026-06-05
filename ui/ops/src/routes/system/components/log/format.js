// Shared formatting helpers for rendering LogMessage.AsObject values,
// used by the full Logs page and the per-service LogsCard.

import {timestampToDate} from '@/api/convpb.js';

export const levelColor = {1: 'grey', 2: 'blue', 3: 'amber', 4: 'red'};
export const levelName = {1: 'DBG', 2: 'INF', 3: 'WRN', 4: 'ERR'};

/**
 * @param {{seconds: number, nanos?: number}|null|undefined} timestamp
 * @return {string}
 */
export function formatTime(timestamp) {
  const date = timestampToDate(timestamp);
  if (!date) return '--:--.---';
  return date.toLocaleTimeString(undefined, {fractionalSecondDigits: 3});
}

/**
 * @param {Array<[string, *]>|undefined} fieldsMap
 * @return {string}
 */
export function formatFields(fieldsMap) {
  if (!fieldsMap?.length) return '';
  return '\t' + JSON.stringify(Object.fromEntries(fieldsMap));
}

/**
 * Splits text into parts, marking case-insensitive occurrences of query.
 * Matching parts get a matchIndex, numbered globally from startIndex so
 * occurrences can be navigated across multiple lines.
 *
 * @param {string} text
 * @param {string} query
 * @param {number} [startIndex]
 * @return {{parts: Array<{text: string, matchIndex?: number}>, count: number}}
 */
export function splitHighlight(text, query, startIndex = 0) {
  if (!query) return {parts: [{text}], count: 0};
  const parts = [];
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  let pos = 0;
  let count = 0;
  for (let i = lower.indexOf(q); i !== -1; i = lower.indexOf(q, pos)) {
    if (i > pos) parts.push({text: text.slice(pos, i)});
    parts.push({text: text.slice(i, i + q.length), matchIndex: startIndex + count});
    count++;
    pos = i + q.length;
  }
  if (pos < text.length) parts.push({text: text.slice(pos)});
  return {parts, count};
}

/**
 * Serialises messages to plain text, one line per message, for export.
 *
 * @param {Array<Object>} messages - LogMessage.AsObject values
 * @return {string}
 */
export function messagesToText(messages) {
  return messages.map(m => {
    const time = timestampToDate(m.timestamp)?.toISOString() ?? '';
    return `${time} ${levelName[m.level] ?? '?'} ${m.logger}: ${m.message}${formatFields(m.fieldsMap)}`;
  }).join('\n') + '\n';
}
