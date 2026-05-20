import {grantNamesByID} from '@/api/sc/traits/access';
import {computed, toValue} from 'vue';

/**
 * Composable for providing status information for access control units.
 *
 * @param {MaybeRefOrGetter<AccessAttempt.AsObject>} accessAttempt
 * @return {{color: import('vue').Ref<string>}}
 */
export function useStatus(accessAttempt) {
  const accessGrantName = computed(() => {
    return grantNamesByID[toValue(accessAttempt)?.grant];
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

  const color = computed(() => accessColor.value ?? 'transparent');

  return {color, accessColor};
}
