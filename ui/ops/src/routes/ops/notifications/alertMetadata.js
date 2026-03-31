import {usePullAlertMetadata} from '@/composables/alerts.js';
import {useCohortStore} from '@/stores/cohort.js';
import {convertProtoMap} from '@/util/proto';
import {defineStore} from 'pinia';
import {computed} from 'vue';

/** @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb').AlertMetadata} AlertMetadata */

export const useAlertMetadataStore = defineStore('alertMetadata', () => {
  const cohort = useCohortStore();
  const name = computed(() => cohort.hubNode?.name ?? '');
  const {value: md, streamError: alertError} = usePullAlertMetadata(name);

  // Return 0 when the total count is not known
  const totalCount = computed(() => (md.value?.totalCount ?? 0));
  const acknowledgedCountMap = computed(() => convertProtoMap(md.value?.acknowledgedCountsMap));
  const resolvedCountMap = computed(() => convertProtoMap(md.value?.resolvedCountsMap));
  const floorCountsMap = computed(() => convertProtoMap(md.value?.floorCountsMap));
  const zoneCountsMap = computed(() => convertProtoMap(md.value?.zoneCountsMap));
  const subsystemCountsMap = computed(() => convertProtoMap(md.value?.subsystemCountsMap));
  const severityCountsMap = computed(() => convertProtoMap(md.value?.severityCountsMap));
  const needsAttentionCountsMap = computed(() => convertProtoMap(md.value?.needsAttentionCountsMap));

  const badgeCount = computed(() => needsAttentionCountsMap.value['nack_unresolved']);
  const unacknowledgedAlertCount = computed(() => acknowledgedCountMap.value[false]);

  return {
    totalCount,
    acknowledgedCountMap,
    resolvedCountMap,
    floorCountsMap,
    zoneCountsMap,
    subsystemCountsMap,
    severityCountsMap,
    needsAttentionCountsMap,

    badgeCount,
    unacknowledgedAlertCount,

    alertError
  };
});
