import {closeResource, newResourceValue} from '@/api/resource.js';
import {listDevices, pullDevices, pullDevicesMetadata} from '@/api/ui/devices.js';
import useFilterCtx from '@/components/filter/filterCtx.js';
import useCollection from '@/composables/collection.js';
import {useExperiment} from '@/composables/experiments.js';
import {watchResource} from '@/util/traits.js';
import {computed, reactive, toRefs, toValue} from 'vue';

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
 * @property {string} search
 *   - if present adds a condition for each word {stringContainsFold: word}
 * @property {Device.Query.Condition.AsObject[]} conditions
 * @property {(value: Device.AsObject, index?: number, array?: Device.AsObject[]) => boolean} filter
 */

/**
 *
 * @param {MaybeRefOrGetter<Partial<UseDevicesOptions>>} props
 * @return {UseCollectionResponse<Device.AsObject> & {
 *   query: import('vue').ComputedRef<Object>,
 * }}
 */
export function useDevices(props) {
  const opts = computed(() => /** @type {Partial<UseDevicesOptions>} */ toValue(props));

  const conditions = computed(() => {
    const _opts = opts.value;
    const conditionsList = [..._opts.conditions ?? []];
    if (_opts.search) {
      const words = _opts.search.split(/\s+/);
      conditionsList.push(...words.map(word => ({stringContainsFold: word})));
    }
    if (_opts.subsystem && _opts.subsystem.toLowerCase() !== 'all') {
      conditionsList.push({field: 'metadata.membership.subsystem', stringEqualFold: _opts.subsystem});
    }
    if (_opts.floor) {
      switch (_opts.floor.toLowerCase()) {
        case 'all':
          // no filter
          break;
        case NO_FLOOR:
          conditionsList.push({field: 'metadata.location.floor', stringEqualFold: ''});
          break;
        default:
          conditionsList.push({field: 'metadata.location.floor', stringEqualFold: _opts.floor});
          break;
      }
    }
    return conditionsList;
  });
  const query = computed(() => {
    return {conditionsList: conditions.value};
  });
  const request = computed(() => {
    return {query: query.value};
  });
  const deviceCollectionOptions = computed(() => {
    return {
      wantCount: opts.value.wantCount ?? 20
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
          cond.stringEqual = 'NORMAL';
        } else {
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
