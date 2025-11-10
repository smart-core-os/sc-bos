import {timestampToDate} from '@/api/convpb.js';
import {closeResource, newResourceCollection} from '@/api/resource.js';
import {pullHealthChecks} from '@/api/sc/traits/health.js';
import {format} from '@/util/number.js';
import {hasOneOf} from '@/util/proto.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * Pull all health checks for a device to get measured values and live updates.
 *
 * @param {import('vue').MaybeRefOrGetter<string|import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').PullHealthChecksRequest.AsObject|null>} request
 * @param {import('vue').MaybeRefOrGetter<boolean>} paused
 * @return {import('vue').ToRefs<ResourceCollection<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject, PullHealthChecksResponse>>}
 */
export function usePullHealthChecks(request, paused = false) {
  const resource = reactive(
      /** @type {ResourceCollection<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject, PullHealthChecksResponse>} */
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
 * @param {Array<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} checks
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.Check.State} state
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
 * @param {Array<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} checks
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
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.Value.AsObject} val
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
