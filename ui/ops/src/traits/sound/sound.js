
/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb').SoundLevel} SoundLevel
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb').SoundLevelSupport} SoundLevelSupport
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb').PullSoundLevelRequest} PullSoundLevelRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb').PullSoundLevelResponse} PullSoundLevelResponse
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb').DescribeSoundLevelRequest} DescribeSoundLevelRequest
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
 */

import {closeResource, newActionTracker, newResourceValue} from '@/api/resource.js';
import {describeSoundLevel, pullSoundLevel} from '@/api/sc/traits/sound-sensor.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullSoundLevelRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<UnwrapNestedRefs<ResourceValue<SoundLevel.AsObject, PullSoundLevelResponse>>>}
 */
export function usePullSoundLevel(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<SoundLevel.AsObject, PullSoundLevelResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullSoundLevel(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<SoundLevel.AsObject>} value
 * @return {{soundPressureLevel: ComputedRef<unknown>}}
 */
export function useSoundLevel(value) {
  const soundPressureLevel = computed(() => {
    const v = toValue(value);
    return v?.soundPressureLevel ?? 0;
  });

  return {
    soundPressureLevel
  };
}

/**
 * @param {MaybeRefOrGetter<string|DescribeSoundLevelRequest.AsObject>} query - The name of the device or a query
 *   object
 * @return {ToRefs<ActionTracker<SoundLevelSupport.AsObject>>}
 */
export function useDescribeSoundLevel(query) {
  const tracker = reactive(
      /** @type {ActionTracker<SoundLevelSupport.AsObject>} */
      newActionTracker()
  );
  
  const queryObject = computed(() => toQueryObject(query));
  
  watchResource(
      () => toValue(queryObject),
      false,
      (req) => {
        describeSoundLevel(req, tracker)
          .catch(() => {}); // errors are tracked by tracker
        return () => closeResource(tracker);
      }
  );
 
 return toRefs(tracker);
}