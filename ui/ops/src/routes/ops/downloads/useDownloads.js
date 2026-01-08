import {ref, toValue, watchEffect} from 'vue';

/**
 * Composable for listing files available for download and downloading them.
 *
 * @param {string} listAllUrl
 * @param {string} downloadUrl
 * @return {{listAvailableFiles: Ref<UnwrapRef<*[]>, UnwrapRef<*[]> | *[]>, downloadFile: downloadFile}}
 */
export function useDownloads(listAllUrl, downloadUrl) {
  listAllUrl = toValue(listAllUrl);
  const listAvailableFiles = ref([]);

  watchEffect(async () => {
    if (!listAllUrl) listAvailableFiles.value = [];

    try {
      const res = await fetch(listAllUrl)

      if (!res.ok) {
        listAvailableFiles.value = [];
        return
      }

      const data = await res.json();

      if (data.count === 0) {
        listAvailableFiles.value = [];
        return;
      }

      listAvailableFiles.value = data.files || [];
    } catch (e) {
      console.error('Failed to fetch available download files', e);
      listAvailableFiles.value = [];
    }
  });

  const downloadFile = (fileUrl) => {
    const url = `${downloadUrl}?file=${encodeURIComponent(fileUrl)}`;
    window.open(url, '_blank');
  }

  return {
    listAvailableFiles,
    downloadFile
  };
}