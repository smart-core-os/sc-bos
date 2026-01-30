import {grantNamesByID} from '@/api/sc/traits/access';
import {StatusLog} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb';
import {computed, toValue} from 'vue';

/**
 * Composable for providing status information for access control units.
 *
 * @param {MaybeRefOrGetter<AccessAttempt.AsObject>} accessAttempt
 * @param {MaybeRefOrGetter<StatusLog.AsObject>} statusLog
 * @return {{
 *  color: import('vue').Ref<string>,
 *  statusColor: import('vue').Ref<string>,
 *  accessColor: import('vue').Ref<string>,
 *  preferStatusLevel: import('vue').Ref<boolean>
 * }}
 */
export function useStatus(accessAttempt, statusLog) {
  const statusLevel = computed(() => {
    return toValue(statusLog)?.level;
  });
  const accessGrantName = computed(() => {
    return grantNamesByID[toValue(accessAttempt)?.grant];
  });

  const preferStatusLevel = computed(() => {
    const level = statusLevel.value;
    return level !== undefined && level >= StatusLog.Level.REDUCED_FUNCTION;
  });
  const statusColor = computed(() => {
    const level = statusLevel.value;
    if (level !== undefined && level >= 0) {
      if (level <= StatusLog.Level.NOMINAL) return 'transparent';
      if (level <= StatusLog.Level.NOTICE) return 'info';
      if (level <= StatusLog.Level.REDUCED_FUNCTION) return 'warning';
      if (level <= StatusLog.Level.NON_FUNCTIONAL) return 'error';
      if (level <= StatusLog.Level.OFFLINE) return 'error';
    }
    return undefined;
  });
  const accessColor = computed(() => {
    const grant = accessGrantName.value?.toLowerCase();
    switch (grant) {
      case 'granted':
      case 'pending':
      case 'aborted':
        return 'success';
      case 'tailgate':
        return 'warning';
      case 'denied':
        return 'error';
      case 'forced':
        return 'error';
      case 'failed':
        return 'error';
    }
    return grant;
  });

  const color = computed(() => {
    const _preferStatusLevel = preferStatusLevel.value;
    const _statusColor = statusColor.value;
    const _accessColor = accessColor.value;

    if (_preferStatusLevel) {
      return _statusColor ?? 'transparent';
    } else {
      return _accessColor ?? _statusColor ?? 'transparent';
    }
  });

  return {
    color,
    statusColor,
    accessColor,
    preferStatusLevel
  };
}
