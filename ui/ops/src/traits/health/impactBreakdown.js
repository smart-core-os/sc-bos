import {usePullDevicesMetadata} from '@/composables/devices.js';
import {useRollingHistory} from '@/traits/health/rollingHistory.js';
import {computed, toValue} from 'vue';

/**
 * @typedef {Object} TableProps
 * @property {string} title
 * @property {number} totalCount
 * @property {string} color
 * @property {string} affectLabel
 * @property {boolean} hideAffected
 * @property {Array<{title: string, count: number, prevCount: number, affect: string}>} issues
 * @property {number} errorCount
 * @property {number} prevErrorCount
 */

/**
 * Returns properties that can be used for a ImpactTable component showing occupant impact health checks.
 *
 * @param {import('vue').MaybeRefOrGetter<Array<{field: string, stringIn?: {stringsList: string[]}}>>} [conditions] - additional conditions to filter devices
 * @return {{
 *   table: import('vue').ComputedRef<TableProps>,
 * }}
 */
export function useOccupantImpactBreakdown(conditions) {
  return useImpactTable(conditions, {
    title: 'People',
    affectLabel: 'People affected',
    impactField: 'occupant_impact',
    fields: [
      {title: 'Life', key: 'LIFE'},
      {title: 'Health', key: 'HEALTH'},
      {title: 'Comfort', key: 'COMFORT'},
    ]
  });
}

/**
 * Returns properties that can be used for a ImpactTable component showing equipment impact health checks.
 *
 * @param {import('vue').MaybeRefOrGetter<Array<{field: string, stringIn?: {stringsList: string[]}}>>} [conditions] - additional conditions to filter devices
 * @return {{
 *   table: import('vue').ComputedRef<TableProps>,
 * }}
 */
export function useEquipmentImpactBreakdown(conditions) {
  return useImpactTable(conditions, {
    title: 'Equipment',
    affectLabel: 'Units affected',
    impactField: 'equipment_impact',
    fields: [
      {title: 'Function', key: 'FUNCTION'},
      {title: 'Warranty', key: 'WARRANTY'},
      {title: 'Lifespan', key: 'LIFESPAN'},
    ]
  });
}

/**
 * Returns properties that can be used for a ImpactTable component showing compliance impact health checks.
 *
 * @param {import('vue').MaybeRefOrGetter<Array<{field: string, stringIn?: {stringsList: string[]}}>>} [conditions] - additional conditions to filter devices
 * @return {{
 *   table: import('vue').ComputedRef<TableProps>,
 * }}
 */
export function useComplianceImpactBreakdown(conditions) {
  return useImpactTable(conditions, {
    title: 'Compliance',
    affectLabel: 'Standards affected',
    impactField: 'compliance_impacts.contribution',
    fields: [
      {title: 'Fail', key: 'FAIL'},
      {title: 'Warning', key: 'WARNING'},
      {title: 'Rating', key: 'RATING'},
      {title: 'Note', key: 'NOTE'},
    ]
  });
}

/**
 * Generic function to create an impact table for a specific type of health check.
 *
 * @param {import('vue').MaybeRefOrGetter<Array<{field: string, stringIn?: {stringsList: string[]}}>>} [conditions]
 * @param {Object} opts
 * @param {string} opts.title - title of the table
 * @param {string} opts.color - color of the score (e.g. 'primary', 'secondary')
 * @param {string} opts.affectLabel - label for the affect column
 * @param {string} opts.impactField - HealthCheck field that represents the impact (e.g. 'equipment_impact')
 * @param {Array<{title: string, key: string}>} opts.fields - fields to include in the table
 * @return {{
 *   table: import('vue').ComputedRef<TableProps>,
 * }}
 */
function useImpactTable(conditions, opts) {
  const impactFieldPath = 'health_checks.' + opts.impactField;
  const totalsQuery = computed(() => {
    const moreConditions = toValue(conditions) ?? [];
    return {
      query: {
        conditionsList: [
          {field: impactFieldPath, stringIn: {stringsList: opts.fields.map(f => f.key)}},
          ...moreConditions
        ],
      },
      includes: {
        fieldsList: [
          impactFieldPath,
          'health_checks.reliability.state',
        ]
      }
    }
  });
  const abnormalQuery = computed(() => {
    const moreConditions = toValue(conditions) ?? [];
    return {
      query: {
        conditionsList: [
          {
            // devices with health checks that are both abnormal and have the specified impact
            field: 'health_checks', matches: {
              conditionsList: [
                {field: 'normality', stringIn: {stringsList: ['ABNORMAL', 'HIGH', 'LOW']}},
                {field: opts.impactField, stringIn: {stringsList: opts.fields.map(f => f.key)}}
              ]
            }
          },
          ...moreConditions
        ],
      },
      includes: {
        fieldsList: [
          impactFieldPath,
        ],
      }
    }
  });
  const {value: totals} = usePullDevicesMetadata(totalsQuery);
  const {oldValue: oldTotals} = useRollingHistory(() => totals.value);
  const {value: abnormals} = usePullDevicesMetadata(abnormalQuery);
  const {oldValue: oldAbnormals} = useRollingHistory(() => abnormals.value);
  const table = computed(() => {
    const oldCounts = getMetadataField(oldAbnormals.value, impactFieldPath)
    const newCounts = getMetadataField(abnormals.value, impactFieldPath)
    const totalCount = getMetadataField(totals.value, impactFieldPath).reduce((acc, curr) => acc + curr[1], 0);
    const issues = opts.fields.map(f => {
      const count = getCountField(newCounts, f.key);
      return {
        title: f.title,
        count,
        prevCount: getCountField(oldCounts, f.key) ?? count,
        affect: '-',
      }
    });
    // todo: fix abnormal counts including checks that aren't matched by the the query
    // The issue is that we are including reliability counts for all checks on devices where some,
    // but not all, of the checks match our base impact query. If there was a device with 2 health checks,
    // one matches our impact query and the other doesn't, the first is reliable and the second is unreliable,
    // our unreliable count will be 1 even though the matching check is reliable.
    const errorCount = totalUnreliableCount(getMetadataField(totals.value, 'health_checks.reliability.state'));
    const prevErrorCount = totalUnreliableCount(getMetadataField(oldTotals.value, 'health_checks.reliability.state'));
    return {
      title: opts.title,
      totalCount,
      affectLabel: opts.affectLabel,
      hideAffected: true,
      issues,
      errorCount,
      prevErrorCount,
    }
  });
  return {
    table
  }
}


/**
 * Returns the counts map for a specific field from the metadata object.
 *
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/devices_pb').DevicesMetadata.AsObject | null} m
 * @param {string} f
 * @return {Array<[string, number]> | null}
 */
function getMetadataField(m, f) {
  return m?.fieldCountsList?.find(r => r.field === f)?.countsMap ?? []
}

/**
 * Returns the count for a specific field from the metadata object.
 *
 * @param {Array<[string, number]> | null} m
 * @param {string} f
 * @return {number}
 */
function getCountField(m, f) {
  for (const [k, v] of m ?? []) {
    if (k === f) {
      return v;
    }
  }
  return 0;
}

/**
 * Returns the total count of unreliable states from a list of [state, count] pairs.
 * The argument is typically the result of a GetDevicesMetadata call.
 *
 * @param {Array<[string, number]>} counts
 * @return {number}
 */
function totalUnreliableCount(counts) {
  return counts?.reduce((acc, [k, v]) => {
    if (k === 'RELIABLE' || k === 'STATE_UNSPECIFIED') return acc;
    return acc + v;
  }, 0);
}
