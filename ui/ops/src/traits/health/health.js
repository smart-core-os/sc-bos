import {timestampToDate} from '@/api/convpb.js';
import {closeResource, newResourceCollection} from '@/api/resource.js';
import {pullHealthChecks} from '@/api/sc/traits/health.js';
import useFilterCtx from '@/components/filter/filterCtx.js';
import {usePullDevicesMetadata, useDevicesMetadataField} from '@/composables/devices.js';
import {format} from '@/util/number.js';
import {hasOneOf} from '@/util/proto.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

const NO_FLOOR = '< no floor >';
const NO_ZONE = '< no zone >';
const NO_SUBSYSTEM = '< no subsystem >';

/**
 * Convert a protobuf enum numeric value to its string name.
 *
 * @param {Object} enumObj - The enum object (e.g., HealthCheck.OccupantImpact)
 * @param {number} value - The numeric enum value
 * @return {string|undefined} - The enum name string (e.g., "COMFORT")
 */
function enumValueToName(enumObj, value) {
  for (const [key, val] of Object.entries(enumObj)) {
    if (val === value) {
      return key;
    }
  }
  return undefined;
}

/**
 * Pull all health checks for a device to get measured values and live updates.
 *
 * @param {import('vue').MaybeRefOrGetter<string|import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').PullHealthChecksRequest.AsObject|null>} request
 * @param {import('vue').MaybeRefOrGetter<boolean>} paused
 * @return {import('vue').ToRefs<ResourceCollection<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject, PullHealthChecksResponse>>}
 */
export function usePullHealthChecks(request, paused = false) {
  const resource = reactive(
      /** @type {ResourceCollection<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject, PullHealthChecksResponse>} */
      newResourceCollection()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(request));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullHealthChecks(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * Counts the number of checks in a specific state.
 *
 * @param {Array<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} checks
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.Check.State} state
 * @return {number}
 */
export function countChecksByNormality(checks, state) {
  return checks?.reduce((acc, check) => {
    if (check.normality === state) acc++;
    return acc;
  }, 0);
}

/**
 * Counts the number of normal and abnormal checks.
 *
 * @param {Array<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} checks
 * @return {{normalCount: number, abnormalCount: number, totalCount: number}}
 */
export function countChecks(checks) {
  const normalCount = countChecksByNormality(checks, HealthCheck.Normality.NORMAL);
  const abnormalCount = checks?.reduce((acc, check) => {
    if (check.normality > HealthCheck.Normality.NORMAL) acc++;
    return acc;
  }, 0);
  return {
    normalCount,
    abnormalCount,
    totalCount: normalCount + abnormalCount,
  }
}

/**
 *
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.Value.AsObject} val
 * @param {string|null} [unit]
 * @return {string}
 */
export function valueToString(val, unit = null) {
  if (hasOneOf(val, 'boolValue')) {
    return `${val.boolValue}`;
  }
  if (hasOneOf(val, 'intValue')) {
    return format(val.intValue, unit);
  }
  if (hasOneOf(val, 'uintValue')) {
    return format(val.uintValue, unit);
  }
  if (hasOneOf(val, 'floatValue')) {
    return format(val.floatValue, unit);
  }
  if (hasOneOf(val, 'stringValue')) {
    return val.stringValue || '-'; // always have a string
  }
  if (hasOneOf(val, 'timestampValue')) {
    return timestampToDate(val.timestampValue).toLocaleString();
  }
  if (hasOneOf(val, 'durationValue')) {
    // todo: better duration formatting
    return format(val.durationValue.seconds, 's');
  }
  return ''; // unknown value
}

/**
 * @param {MaybeRefOrGetter<Record<string, any>>?} forcedFilters
 * @return {{
 *   filterOpts: import('vue').Ref<import('@/components/filter/filterCtx').Options>,
 *   filterCtx: import('@/components/filter/filterCtx').FilterCtx,
 *   forcedConditions: import('vue').ComputedRef<import('@smart-core-os/sc-bos-ui-gen/proto/devices_pb').Device.Query.Condition.AsObject[]>,
 *   filterConditions: import('vue').ComputedRef<import('@smart-core-os/sc-bos-ui-gen/proto/devices_pb').Device.Query.Condition.AsObject[]>,
 * }}
 */
export function useHealthCheckFilters(forcedFilters) {
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

    if (!Object.hasOwn(forced, 'health_checks.occupant_impact')) {
      filters.push({
        key: 'health_checks.occupant_impact',
        icon: 'mdi-account-group',
        title: 'Occupant Impact',
        type: 'list',
        items: [
          {title: 'No Impact', value: HealthCheck.OccupantImpact.NO_OCCUPANT_IMPACT},
          {title: 'Comfort', value: HealthCheck.OccupantImpact.COMFORT},
          {title: 'Health', value: HealthCheck.OccupantImpact.HEALTH},
          {title: 'Life Safety', value: HealthCheck.OccupantImpact.LIFE},
          {title: 'Security', value: HealthCheck.OccupantImpact.SECURITY}
        ]
      });
    }

    if (!Object.hasOwn(forced, 'health_checks.equipment_impact')) {
      filters.push({
        key: 'health_checks.equipment_impact',
        icon: 'mdi-tools',
        title: 'Equipment Impact',
        type: 'list',
        items: [
          {title: 'No Impact', value: HealthCheck.EquipmentImpact.NO_EQUIPMENT_IMPACT},
          {title: 'Warranty', value: HealthCheck.EquipmentImpact.WARRANTY},
          {title: 'Lifespan', value: HealthCheck.EquipmentImpact.LIFESPAN},
          {title: 'Function', value: HealthCheck.EquipmentImpact.FUNCTION}
        ]
      });
    }

    return {filters, defaults};
  });

  const filterCtx = useFilterCtx(filterOpts);

  const toCondition = (field, value) => {
    if (value == null) return null;
    switch (field) {
      case 'metadata.location.floor':
        return {field: 'metadata.location.floor', stringEqualFold: value === NO_FLOOR ? '' : value};
      case 'metadata.location.zone':
        return {field: 'metadata.location.zone', stringEqualFold: value === NO_ZONE ? '' : value};
      case 'metadata.membership.subsystem':
        return {field: 'metadata.membership.subsystem', stringEqualFold: value === NO_SUBSYSTEM ? '' : value};
      case 'health_checks.occupant_impact': {
        const numVal = value?.value ?? value;
        const enumName = enumValueToName(HealthCheck.OccupantImpact, numVal);
        return {field: 'health_checks.occupant_impact', stringEqual: enumName};
      }
      case 'health_checks.equipment_impact': {
        const numVal = value?.value ?? value;
        const enumName = enumValueToName(HealthCheck.EquipmentImpact, numVal);
        return {field: 'health_checks.equipment_impact', stringEqual: enumName};
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

