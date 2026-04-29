import {timestampToDate} from '@/api/convpb.js';
import {getDownloadDevicesUrl} from '@/api/ui/devices.js';

import {MINUTE} from '@/util/date.js';
import {computed, onScopeDispose, ref, toValue, watch} from 'vue';

/**
 * @param {import('vue').MaybeRefOrGetter<Partial<Device.Query.AsObject>>} query
 * @param {import('vue').MaybeRefOrGetter<Partial<Period.AsObject|undefined|null>>} [history]
 * @return {{
 *   downloadBtnProps: ComputedRef<Record<string,any>>
 * }}
 */
export function useDownloadLink(query, history = null) {
  const tableDownloadUrl = ref(/** @type {DownloadDevicesUrl.AsObject | null} */ null);
  const getDownloadDevicesUrlRequest = computed(() => {
    return {query: toValue(query), history: toValue(history)};
  });

  const leeway = 1 * MINUTE;
  let refreshHandle = 0;
  onScopeDispose(() => clearTimeout(refreshHandle));

  const fetchDownloadUrl = async (request) => {
    clearTimeout(refreshHandle);
    if (!request) return;
    try {
      const url = await getDownloadDevicesUrl(request);
      tableDownloadUrl.value = url;
      if (url.expireAfterTime) {
        const expireAtDate = timestampToDate(url.expireAfterTime);
        const expireIn = expireAtDate - Date.now() - leeway;
        if (expireIn > 0) {
          refreshHandle = setTimeout(() => fetchDownloadUrl(request), expireIn);
        }
      }
    } catch (e) {
      console.warn('Failed to get download devices URL', e);
    }
  };

  watch(getDownloadDevicesUrlRequest, async (request) => {
    tableDownloadUrl.value = null;
    await fetchDownloadUrl(request);
  }, {immediate: true, deep: true});

  const downloadBtnProps = computed(() => {
    const props = {};
    const url = tableDownloadUrl.value;
    if (url) {
      props.href = url.url;
      props.download = url.filename;
    }
    props.disabled = !url;

    return props;
  });

  return {
    downloadBtnProps
  };
}

/**
 * Converts a time period into a string suitable for use in a filename.
 *
 * @example
 * datePeriodString({startTime: new Date(2021, 0, 1), endTime: new Date(2021, 0, 2)}) // '2021-01-01-to-2021-01-02'
 * datePeriodString({startTime: new Date(2021, 0, 1), endTime: new Date(2021, 1, 1)}) // '2021-01-to-2021-02'
 * datePeriodString({startTime: new Date(2021, 0, 1), endTime: new Date(2022, 0, 1)}) // '2021-to-2022'
 *
 * @param {Period.AsObject | {startTime: Date, endTime: Date} | null | undefined} period
 * @return {string}
 */
function datePeriodString(period) {
  const toStr = (n) => n.toString().padStart(2, '0');
  if (period) {
    const start = timestampToDate(period.startTime);
    const end = timestampToDate(period.endTime);
    let s = '';
    if (start.getFullYear() === end.getFullYear()) {
      s += `${start.getFullYear()}-`;
    } else if (start.getMonth() === end.getMonth()) {
      return `${start.getFullYear()}-to-${end.getFullYear()}`;
    } else {
      return `${start.getFullYear()}-${toStr(start.getMonth() + 1)}-to-${end.getFullYear()}-${toStr(end.getMonth() + 1)}`;
    }
    if (start.getMonth() === end.getMonth()) {
      s += `${toStr(start.getMonth() + 1)}-`;
    } else if (start.getDate() === end.getDate()) {
      return `${s}${toStr(start.getMonth() + 1)}-to-${toStr(end.getMonth() + 1)}`;
    } else {
      return `${s}${toStr(start.getMonth() + 1)}-${toStr(start.getDate())}-to-${toStr(end.getMonth() + 1)}-${toStr(end.getDate())}`;
    }
    if (start.getDate() === end.getDate()) {
      s += `${toStr(start.getDate())}`;
    } else {
      return `${s}${toStr(start.getDate())}-to-${toStr(end.getDate())}`;
    }
    return s;
  }
  const date = new Date();
  return `${date.getFullYear()}-${toStr(date.getMonth() + 1)}-${toStr(date.getDate())}`;
}

/**
 * Call as part of a user interaction (i.e. click) to trigger the browser to download the devices associated with the given query.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} [qual] - filename qualifier
 * @param {import('vue').MaybeRefOrGetter<Device.Query.AsObject>} query - query for devices
 * @param {import('vue').MaybeRefOrGetter<null | Period.AsObject>} [history] - a period to fetch historical records for
 * @param {import('vue').MaybeRefOrGetter<null | GetDownloadDevicesUrlRequest.Table.AsObject>} [table] - options for the table
 * @return {Promise<void>}
 */
export async function triggerDownload(qual = 'devices', query, history = null, table = null) {
  const _history = toValue(history);

  const dateString = datePeriodString(_history);
  const filename = `${toValue(qual)}-${dateString}.csv`;

  const urlRes = await getDownloadDevicesUrl({
    filename,
    query: toValue(query),
    history,
    table: toValue(table),
  });

  triggerDownloadFromUrl(urlRes.url, filename);
}

/**
 * Triggers a browser download for the given URL and filename.
 *
 * @param {string} url
 * @param {string} filename
 */
export function triggerDownloadFromUrl(url, filename) {
  const a = document.createElement('a');
  a.style.position = 'absolute';
  a.style.top = '0';
  a.style.visibility = 'hidden';
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}
