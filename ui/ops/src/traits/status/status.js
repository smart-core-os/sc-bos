import {closeResource, newResourceValue} from '@/api/resource';
import {pullCurrentStatus} from '@/api/sc/traits/status';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {StatusLog} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb').StatusLog} StatusLog
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb').PullCurrentStatusRequest} PullCurrentStatusRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb').PullCurrentStatusResponse} PullCurrentStatusResponse
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 */

/**
 * @param {MaybeRefOrGetter<string|PullCurrentStatusRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<StatusLog.AsObject, PullCurrentStatusResponse>>}
 */
export function usePullCurrentStatus(query, paused = false) {
  const statusValue = reactive(
      /** @type {ResourceValue<StatusLog.AsObject, StatusLog>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(statusValue));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullCurrentStatus(req, statusValue);
        return () => closeResource(statusValue);
      }
  );

  return toRefs(statusValue);
}

/**
 * @param {MaybeRefOrGetter<StatusLog.AsObject>} value
 * @return {{
 *   level: ComputedRef<StatusLog.Level>,
 *   levelToStr: (level: StatusLog.Level, prefix?: string) => string,
 *   iconStr: ComputedRef<string>,
 *   iconColor: ComputedRef<string>,
 *   description: ComputedRef<string>,
 *   hasMoreProblems: ComputedRef<boolean>,
 *   levelStr: ComputedRef<string>,
 *   ok: ComputedRef<boolean>,
 *   notOk: ComputedRef<boolean>,
 *   problems: ComputedRef<StatusLog.Problem.AsObject[]>
 * }}
 */
export function useStatusLog(value) {
  const _v = computed(() => toValue(value));

  const level = computed(() => _v.value?.level ?? 0);
  const levelToStr = (level, prefix = '') => {
    if (level === StatusLog.Level.LEVEL_UNDEFINED) return '';
    if (level <= StatusLog.Level.NOMINAL) return prefix + 'Nominal';
    if (level <= StatusLog.Level.NOTICE) return prefix + 'Notice';
    if (level <= StatusLog.Level.REDUCED_FUNCTION) return prefix + 'Reduced Function';
    if (level <= StatusLog.Level.NON_FUNCTIONAL) return prefix + 'Non-Functional';
    if (level <= StatusLog.Level.OFFLINE) return prefix + 'Offline';
    return 'Custom Level ' + level;
  };
  const levelStr = computed(() => levelToStr(level.value));
  const description = computed(() => _v.value?.description ?? '');

  const iconColor = computed(() => {
    if (level.value <= StatusLog.Level.NOTICE) return 'info';
    if (level.value <= StatusLog.Level.REDUCED_FUNCTION) return 'warning';
    if (level.value <= StatusLog.Level.NON_FUNCTIONAL) return 'error';
    if (level.value <= StatusLog.Level.OFFLINE) return 'grey';
    if (level.value <= StatusLog.Level.NOMINAL) return 'success';
    return 'white';
  });
  const iconStr = computed(() => {
    if (level.value <= StatusLog.Level.NOMINAL) return 'mdi-check-circle-outline';
    if (level.value <= StatusLog.Level.NOTICE) return 'mdi-information-outline';
    if (level.value <= StatusLog.Level.REDUCED_FUNCTION) return 'mdi-progress-alert';
    if (level.value <= StatusLog.Level.NON_FUNCTIONAL) return 'mdi-alert-circle-outline';
    if (level.value <= StatusLog.Level.OFFLINE) return 'mdi-connection';
    return '';
  });

  const problems = computed(() => _v.value?.problemsList ?? []);
  const hasMoreProblems = computed(() => problems.value.length > 0);

  // note, these aren't mutually exclusive. props.value === null will be false for both for example
  const ok = computed(() => level.value === StatusLog.Level.NOMINAL);
  const notOk = computed(() => level.value > StatusLog.Level.NOMINAL);

  return {
    level,
    levelToStr,
    levelStr,
    description,
    iconColor,
    iconStr,
    problems,
    hasMoreProblems,
    ok,
    notOk
  };
}
