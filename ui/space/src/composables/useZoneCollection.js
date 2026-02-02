import {closeResource, newActionTracker} from '@/api/resource';
import {getEnrollment} from '@/api/sc/traits/enrollment';
import {listHubNodes} from '@/api/sc/traits/hub';
import {listServices} from '@/api/ui/services';
import {computed, onMounted, onUnmounted, reactive, ref, watch} from 'vue';

/**
 * @return {{
 *  hubNodesValue: import('@/api/resource').ActionTracker,
 *  enrollmentValue: import('@/api/resource').ActionTracker,
 *  allNamesToCheck: import('vue').ComputedRef<string[]>,
 *  zoneCollection: import('@/api/resource').ActionTracker
 * }}
 */
export default function() {
  const hubNodesValue = reactive(/** @type {ActionTracker<ListHubNodesResponse.AsObject>} */ newActionTracker());
  const enrollmentValue = reactive(/** @type {ActionTracker<GetEnrollmentResponse.AsObject>} */ newActionTracker());
  const zoneCollection = reactive(/** @type {ActionTracker<ListServicesResponse.AsObject>} */ newActionTracker());
  const loadNextPage = ref(false);

  const allNamesToCheck = computed(() => {
    // Find all or fill in the hub node names
    const hubNodeNames = hubNodesValue?.response?.nodesList ?
        hubNodesValue.response.nodesList.map(n => n.name || '') :
        [''];

    // Find or fill in the manager and target names
    const managerName = enrollmentValue?.response?.managerName || '';
    const targetName = enrollmentValue?.response?.targetName || '';

    // Combine all names and filter out duplicates using Set
    return [...new Set([...hubNodeNames, managerName, targetName])];
  });
  const namesVersion = ref(0); // used to make sure responses should be honoured or ignored

  // cursor holds the parameters for the next request we should perform
  const cursor = ref(
      {nameIndex: 0, pageToken: ''}
  );

  /**
   *
   * @param {Partial<ListServicesRequest.AsObject>} baseRequest
   * @return {number} the number of zones fetched. Anything other than null means there may be more to fetch
   */
  async function listNextZones(baseRequest) {
    if (cursor.value.nameIndex >= allNamesToCheck.value.length) {
      // We have already checked all names
      return null;
    }

    const name = allNamesToCheck.value[cursor.value.nameIndex];
    const req = {
      ...baseRequest,
      name: name === '' ? 'zones' : name + '/zones',
      pageToken: cursor.value.pageToken
    };

    zoneCollection.loading = true;
    const v0 = namesVersion.value;
    try {
      const res = await listServices(req);
      if (v0 !== namesVersion.value) {
        return 0; // ignore this response
      }

      // setup the next page to be loaded
      cursor.value.pageToken = res.nextPageToken;
      if (cursor.value.pageToken === '') {
        cursor.value.nameIndex++;
      }

      if (!zoneCollection.response) {
        zoneCollection.response = res;
      } else {
        zoneCollection.response.servicesList.push(...res.servicesList);
        zoneCollection.response.totalSize += res.totalSize;
      }

      return res.servicesList?.length || 0;
    } catch (e) {
      if (v0 !== namesVersion.value) {
        return 0; // ignore this response
      }
      zoneCollection.error = e;
    } finally {
      if (v0 === namesVersion.value) {
        zoneCollection.loading = false;
      }
    }
    // if we get this far, we need to try another name if one is available
    cursor.value.nameIndex++;
    if (cursor.value.nameIndex >= allNamesToCheck.value.length) {
      return null;
    }
    return 0;
  }

  const getNextZones = (pageSize) => {
    if (pageSize === 0) {
      return;
    }
    const baseRequest = {
      pageSize: pageSize
    };
    // errors are handled by tracker
    return listNextZones(baseRequest);
  };

  watch(allNamesToCheck, () => {
    cursor.value = {nameIndex: 0, pageToken: ''};
    namesVersion.value++;
  });
  watch([allNamesToCheck, loadNextPage], ([names, loadPage]) => {
    if (!loadPage || names.length === 0) {
      return;
    }
    getNextZones(100).catch(() => {
      // handled by tracker
    });
  }, {deep: true, immediate: true});


  onMounted(() => {
    listHubNodes(hubNodesValue).catch(() => {
      /* handled by tracker */
    });
    getEnrollment(enrollmentValue).catch(() => {
      /* handled by tracker */
    });
  });

  onUnmounted(() => {
    closeResource(hubNodesValue);
    closeResource(enrollmentValue);
    closeResource(zoneCollection);
  });

  return {
    hubNodesValue,
    enrollmentValue,
    zoneCollection,

    loadNextPage,
    getNextZones,

    allNamesToCheck,
    cursor
  };
}
