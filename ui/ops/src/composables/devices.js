import {closeResource, newActionTracker, newResourceValue} from '@/api/resource.js';
import {getDevicesMetadata, listDevices, pullDevices, pullDevicesMetadata} from '@/api/ui/devices.js';
import useFilterCtx from '@/components/filter/filterCtx.js';
import useCollection from '@/composables/collection.js';
import {useExperiment} from '@/composables/experiments.js';
import {usePoll} from '@/composables/poll.js';
import {SECOND} from '@/util/date.js';
import {watchResource} from '@/util/traits.js';
import {Device} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/devices/v1/devices_pb';
import {computed, reactive, toRefs, toValue, watch} from 'vue';

/**
 * @param {MaybeRefOrGetter<Partial<ListDevicesRequest.AsObject>>} request
 * @param {MaybeRefOrGetter<Partial<UseCollectionOptions>>?} options
 * @return {UseCollectionResponse<Device.AsObject>}
 */
export function useDevicesCollection(request, options) {
  const normOptions = computed(() => {
    const optArg = toValue(options);
    return {
      cmp: (a, b) => a.name.localeCompare(b.name),
      ...optArg
    };
  });
  const client = {
    async listFn(req, tracker) {
      const res = await listDevices(req, tracker);
      return {
        items: res.devicesList,
        nextPageToken: res.nextPageToken,
        totalSize: res.totalSize
      };
    },
    pullFn(req, resource) {
      pullDevices(req, resource);
    }
  };
  return useCollection(request, client, normOptions);
}

/**
 * @param {import('vue').MaybeRefOrGetter<string|string[]|PullDevicesMetadataRequest.AsObject>} query
 * @param {import('vue').MaybeRefOrGetter<{paused?: boolean}>?} options
 * @return {import('vue').ToRefs<ResourceValue<DevicesMetadata.AsObject, PullDevicesMetadataResponse>>}
 */
export function usePullDevicesMetadata(query, options) {
  const normQuery = computed(() => {
    const queryArg = toValue(query);
    if (typeof queryArg === 'string') {
      return {includes: {fieldsList: [queryArg]}};
    }
    if (Array.isArray(queryArg)) {
      return {includes: {fieldsList: queryArg}};
    }
    // we could check for the correct type here, but lets assume people know what they're doing
    return queryArg;
  });

  const resource = reactive(
      /** @type {ResourceValue<DevicesMetadata.AsObject, PullDevicesMetadataResponse>} */
      newResourceValue());

  watchResource(normQuery, () => toValue(options)?.paused ?? false, (req) => {
    pullDevicesMetadata(req, resource);
    return () => closeResource(resource);
  });

  return toRefs(resource);
}

/**
 * @param {import('vue').MaybeRefOrGetter<DevicesMetadata.AsObject>} value
 * @param {import('vue').MaybeRefOrGetter<string>} field
 * @return {{
 *  counts: import('vue').Ref<Array<[string, number]>>,
 *  countsMap: import('vue').Ref<Record<string, number>>,
 *  keys: import('vue').Ref<string[]>
 * }}
 */
export function useDevicesMetadataField(value, field) {
  const counts = computed(() => {
    const _value = toValue(value);
    const _field = toValue(field);
    return _value?.fieldCountsList?.find(v => v.field === _field)?.countsMap;
  });
  const countMap = computed(() => {
    const mapArr = counts.value || [];
    if (mapArr.length === 0) return {};
    return mapArr.reduce((acc, [k, v]) => {
      acc[k] = v;
      return acc;
    }, {});
  });
  const keys = computed(() => {
    return (counts.value ?? []).map(([k]) => k);
  });

  return {
    counts,
    countMap,
    keys
  };
}

const NO_FLOOR = '< no floor >';
const NO_ZONE = '< no zone >';
const NO_SUBSYSTEM = '< no subsystem >';

/**
 * @typedef {Object} UseDevicesOptions
 * @property {number} wantCount
 * @property {string} subsystem
 *   - if present and not 'all', adds the condition {field: "metadata.membership.subsystem", stringEqualFold: subsystem}
 * @property {string} floor
 *   - if present and not 'all', adds the condition {field: "metadata.location.floor", stringEqualFold: floor}
 * @property {string|string[]} trait
 *   - a fully-qualified trait name, or several; matches devices implementing it/any of them
 * @property {string} search
 *   - if present adds a condition for each word {stringContainsFold: word}
 * @property {Device.Query.Condition.AsObject[]} conditions
 * @property {(value: Device.AsObject, index?: number, array?: Device.AsObject[]) => boolean} filter
 */

/**
 * Builds the query conditions described by opts. Shared by useDevices and
 * useDeviceHealthCount so both select the same devices from the same props.
 *
 * @param {Partial<UseDevicesOptions>} opts
 * @return {Device.Query.Condition.AsObject[]}
 */
export function deviceConditions(opts) {
  const conditionsList = [...opts.conditions ?? []];
  if (opts.search) {
    const words = opts.search.split(/\s+/);
    conditionsList.push(...words.map(word => ({stringContainsFold: word})));
  }
  if (opts.subsystem && opts.subsystem.toLowerCase() !== 'all') {
    conditionsList.push({field: 'metadata.membership.subsystem', stringEqualFold: opts.subsystem});
  }
  if (opts.floor) {
    switch (opts.floor.toLowerCase()) {
      case 'all':
        // no filter
        break;
      case NO_FLOOR:
        conditionsList.push({field: 'metadata.location.floor', stringEqualFold: ''});
        break;
      default:
        conditionsList.push({field: 'metadata.location.floor', stringEqualFold: opts.floor});
        break;
    }
  }
  if (opts.trait) {
    // Trait names are protobuf identifiers, so they're matched exactly. The other branches
    // fold case because subsystems and floors are free text typed by an integrator.
    const traits = Array.isArray(opts.trait) ? opts.trait : [opts.trait];
    if (traits.length === 1) {
      conditionsList.push({field: 'metadata.traits.name', stringEqual: traits[0]});
    } else if (traits.length > 1) {
      // A list reads as "implements any of these": metadata.traits.name is a value set and the
      // query Matcher defaults to ANY. One condition per trait would AND them instead, as
      // conditionsList entries are conjunctive.
      conditionsList.push({field: 'metadata.traits.name', stringIn: {stringsList: traits}});
    }
  }
  return conditionsList;
}

/**
 *
 * @param {MaybeRefOrGetter<Partial<UseDevicesOptions>>} props
 * @return {UseCollectionResponse<Device.AsObject> & {
 *   query: import('vue').ComputedRef<Object>,
 * }}
 */
export function useDevices(props) {
  const opts = computed(() => /** @type {Partial<UseDevicesOptions>} */ toValue(props));

  const conditions = computed(() => deviceConditions(opts.value));
  const query = computed(() => {
    return {conditionsList: conditions.value};
  });
  const request = computed(() => {
    return {query: query.value};
  });
  const deviceCollectionOptions = computed(() => {
    return {
      wantCount: opts.value.wantCount ?? 20,
      paused: opts.value.paused ?? false,
    };
  });

  const collection = useDevicesCollection(request, deviceCollectionOptions);

  // Computed property for the filtered table data
  const items = computed(() => {
    const values = collection.items.value;
    if (!opts.value.filter) return values;
    return values.filter(opts.value.filter);
  });

  return {
    ...collection,
    query,
    items
  };
}

/**
 * Normality values that mean a health check is reporting a problem.
 *
 * @type {string[]}
 */
export const ABNORMAL_NORMALITIES = ['ABNORMAL', 'HIGH', 'LOW'];

/**
 * Reliability states that mean a health check couldn't be read.
 *
 * @type {string[]}
 */
export const UNRELIABLE_STATES = [
  'UNRELIABLE', 'CONN_TRANSIENT_FAILURE', 'SEND_FAILURE', 'NO_RESPONSE',
  'BAD_RESPONSE', 'NOT_FOUND', 'PERMISSION_DENIED'
];

/**
 * Query conditions matching devices that have an issue: any health check that is either
 * abnormal or unreadable.
 *
 * Both dimensions are needed. They move independently - a comms failure sets reliability
 * and leaves normality NORMAL, while a fault sets normality and leaves reliability
 * RELIABLE - so checking one and not the other misses half the failures. any_of ORs them
 * inside a single condition, so the two still have to hold of the *same* check. See the
 * Example_ health query in internal/manage/devices/query_test.go, which this mirrors.
 *
 * Note this is stricter than the Health Status filter in useDeviceFilters, which looks at
 * normality alone.
 *
 * @return {Device.Query.Condition.AsObject[]}
 */
export function unhealthyDeviceConditions() {
  return [{
    field: 'health_checks',
    // Matcher defaults to ANY: the device matches if any health check matches anyOf.
    anyOf: {
      queriesList: [
        {conditionsList: [{field: 'normality', stringIn: {stringsList: ABNORMAL_NORMALITIES}}]},
        {conditionsList: [{field: 'reliability.state', stringIn: {stringsList: UNRELIABLE_STATES}}]}
      ]
    }
  }];
}

/**
 * @typedef {UseDevicesOptions} UseDeviceHealthCountOptions
 * @property {number} expected
 *   - if > 0, the total to report instead of the live device count. A live count makes an
 *     outage invisible: when a node drops off the cohort its devices are removed, shrinking
 *     the numerator and the denominator together. Pin this to the number of devices that
 *     are supposed to exist and a whole node going dark reads as 0 / n instead.
 * @property {number} pollPeriod - how often to recount, in milliseconds.
 */

/**
 * Counts how many of the devices matching props are reporting normally.
 *
 * Both figures are counted by the server, as DevicesMetadata.total_count over the matching
 * and the unhealthy queries. Nothing about the devices themselves is transferred, so this
 * costs the same for ten devices as for a thousand.
 *
 * total_count is the right field for this; the field_counts in the same message are not.
 * Those bucket a repeated field once per unique value, so a device with one NORMAL and one
 * ABNORMAL check lands in both buckets and the counts don't add up to the total.
 *
 * Polled rather than pulled. Two counts need two queries, and a stream apiece would mean two
 * held connections per caller: put four of these on a dashboard and they exhaust the
 * browser's six-per-origin connection limit between them, leaving the later ones showing
 * nothing at all. A coverage percentage does not need sub-second liveness, so this trades it
 * for composability.
 *
 * @param {MaybeRefOrGetter<Partial<UseDeviceHealthCountOptions>>} props
 * @return {{
 *   total: import('vue').ComputedRef<number>,
 *   unhealthy: import('vue').ComputedRef<number>,
 *   online: import('vue').ComputedRef<number>,
 *   percent: import('vue').ComputedRef<number>,
 *   hasData: import('vue').ComputedRef<boolean>,
 *   loading: import('vue').ComputedRef<boolean>,
 *   error: import('vue').ComputedRef<RemoteError|null>,
 *   refresh: (force?: boolean) => void
 * }}
 */
export function useDeviceHealthCount(props) {
  const opts = computed(() => /** @type {Partial<UseDeviceHealthCountOptions>} */ toValue(props) ?? {});
  const conditions = computed(() => deviceConditions(opts.value));

  const allTracker = reactive(/** @type {ActionTracker<DevicesMetadata.AsObject>} */ newActionTracker());
  const unhealthyTracker = reactive(/** @type {ActionTracker<DevicesMetadata.AsObject>} */ newActionTracker());

  const count = async () => {
    if (opts.value.paused) return;
    const conditionsList = conditions.value;
    // Counted together so the two figures describe the same moment as closely as we can
    // manage; online is their difference, and a stale total against a fresh unhealthy count
    // can read as more devices broken than exist.
    await Promise.all([
      // the trackers record the errors, and one failing shouldn't abandon the other
      getDevicesMetadata({query: {conditionsList}}, allTracker).catch(() => {}),
      getDevicesMetadata({
        query: {conditionsList: [...conditionsList, ...unhealthyDeviceConditions()]}
      }, unhealthyTracker).catch(() => {})
    ]);
  };

  const {pollNow, isPolling} = usePoll(count, () => opts.value.pollPeriod ?? 30 * SECOND);
  // recount straight away when the query changes rather than waiting out the period
  watch(conditions, () => pollNow(true), {deep: true});

  const liveTotal = computed(() => allTracker.response?.totalCount ?? 0);
  const total = computed(() => {
    const expected = opts.value.expected;
    return expected > 0 ? expected : liveTotal.value;
  });
  const unhealthy = computed(() => unhealthyTracker.response?.totalCount ?? 0);
  // Devices missing from the live list are offline too, which is the whole point of
  // expected: they are counted here by never being added to online in the first place.
  const online = computed(() => Math.max(0, Math.min(total.value, liveTotal.value - unhealthy.value)));
  // No devices reads as 0%, not 100%: there is nothing to be confident about.
  const percent = computed(() => total.value === 0 ? 0 : (online.value / total.value) * 100);

  // Keyed on the matching-devices count alone. Both are needed for a figure, but a count of
  // zero is a real answer, so waiting for both would be waiting for nothing in the case we
  // most expect: a healthy system, where the unhealthy count is legitimately 0.
  const hasData = computed(() => Boolean(allTracker.response));
  const loading = computed(() => isPolling.value && !hasData.value);
  const error = computed(() => allTracker.error ?? unhealthyTracker.error ?? null);

  return {total, unhealthy, online, percent, hasData, loading, error, refresh: pollNow};
}

/**
 * Returns the list of floors, suitable for use in a select box.
 * Each item in the floorList is suitable for use as the `floor` prop in the useDevices function.
 *
 * @return {{floorList: ComputedRef<string[]>}}
 */
export function useDeviceFloorList() {
  const {value: md} = usePullDevicesMetadata('metadata.location.floor');
  const {keys: listOfFloors} = useDevicesMetadataField(md, 'metadata.location.floor');
  const floorList = computed(() => {
    return ['All', ...listOfFloors.value
        .sort((a, b) => a.localeCompare(b, undefined, {numeric: true}))
        .map(v => v === '' ? NO_FLOOR : v)];
  });
  return {floorList};
}

/**
 * @param {MaybeRefOrGetter<Record<string, any>>?} forcedFilters
 * @return {{
 *   filterOpts: Ref<import('@/components/filter/filterCtx').Options>,
 *   filterCtx: import('@/components/filter/filterCtx').FilterCtx,
 *   forcedConditions: import('vue').Ref<Device.Query.Condition.AsObject[]>,
 *   filterConditions: import('vue').Ref<Device.Query.Condition.AsObject[]>,
 * }}
 */
export function useDeviceFilters(forcedFilters) {
  const healthExperiment = useExperiment('health');

  const {value: md} = usePullDevicesMetadata([
    'metadata.location.floor',
    'metadata.location.zone',
    'metadata.membership.subsystem'
  ]);
  const {keys: floorKeys} = useDevicesMetadataField(md, 'metadata.location.floor');
  const {keys: zoneKeys} = useDevicesMetadataField(md, 'metadata.location.zone');
  const {keys: subsystemKeys} = useDevicesMetadataField(md, 'metadata.membership.subsystem');
  const filterOpts = computed(() => {
    const filters = [];
    const defaults = [];

    const forced = toValue(forcedFilters) ?? {};

    if (!Object.hasOwn(forced, 'metadata.location.floor')) {
      const floors = [...floorKeys.value]
          .sort((a, b) => a.localeCompare(b, undefined, {numeric: true}))
          .map(f => f === '' ? NO_FLOOR : f);
      if (floors.length > 1) {
        filters.push({
          key: 'metadata.location.floor',
          icon: 'mdi-layers-triple-outline',
          title: 'Floor',
          type: 'list',
          items: floors
        });
      }
    }

    if (!Object.hasOwn(forced, 'metadata.location.zone')) {
      const zones = zoneKeys.value.map(z => z === '' ? NO_ZONE : z);
      if (zones.length > 1) {
        filters.push({
          key: 'metadata.location.zone',
          icon: 'mdi-select-all',
          title: 'Zone',
          type: 'list',
          items: zones
        });
      }
    }

    if (!Object.hasOwn(forced, 'metadata.membership.subsystem')) {
      const subsystems = subsystemKeys.value.map(s => s === '' ? NO_SUBSYSTEM : s);
      if (subsystems.length > 1) {
        filters.push({
          key: 'metadata.membership.subsystem',
          icon: 'mdi-cube-outline',
          title: 'Subsystem',
          type: 'list',
          items: subsystems
        });
      }
    }

    if (healthExperiment.value) {
      if (!Object.hasOwn(forced, 'health_checks.normality')) {
        filters.push({
          key: 'health_checks.normality',
          icon: 'mdi-heart-pulse',
          title: 'Health Status',
          type: 'boolean',
          valueToString(value) {
            switch (value) {
              case true:
                return 'Healthy';
              case false:
                return 'Unhealthy';
              default:
                return 'All';
            }
          }
        })
      }
    }

    return {filters, defaults};
  });

  const filterCtx = useFilterCtx(filterOpts);

  const toCondition = (field, value) => {
    if (value === undefined || value === null) return null;
    switch (field) {
      case 'floor':
      case 'metadata.location.floor':
        return {field: 'metadata.location.floor', stringEqualFold: value === NO_FLOOR ? '' : value};
      case 'zone':
      case 'metadata.location.zone':
        return {field: 'metadata.location.zone', stringEqualFold: value === NO_ZONE ? '' : value};
      case 'subsystem':
      case 'metadata.membership.subsystem':
        return {field: 'metadata.membership.subsystem', stringEqualFold: value === NO_SUBSYSTEM ? '' : value};
      case 'health_checks.normality': {
        const cond = {field: 'health_checks.normality'};
        if (value) {
          // all checks should be normal to be classed as healthy
          cond.matcher = Device.Query.Condition.Matcher.ALL;
          cond.stringEqual = 'NORMAL';
        } else {
          // any check should be abnormal to be classed as unhealthy
          cond.stringIn = {stringsList: ['ABNORMAL', 'HIGH', 'LOW']}
        }
        return cond;
      }
      default:
        return {field: field, stringEqualFold: value};
    }
  };

  const forcedConditions = computed(() => {
    const res = [];
    for (const [k, v] of Object.entries(toValue(forcedFilters) ?? {})) {
      const cond = toCondition(k, v);
      if (cond) res.push(cond);
    }
    return res;
  });
  const filterConditions = computed(() => {
    const res = [];
    const choices = /** @type {import('@/components/filter/filterCtx').Choice[]} */ filterCtx.sortedChoices.value;
    for (const choice of choices) {
      const cond = toCondition(choice?.filter, choice?.value);
      if (cond) res.push(cond);
    }
    return res;
  });

  return {filterOpts, filterCtx, forcedConditions, filterConditions};
}
