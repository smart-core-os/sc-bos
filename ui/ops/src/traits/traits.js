import {usePullAccessAttempts} from '@/traits/access/access.js';
import {usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {usePullAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {usePullElectricDemand} from '@/traits/electricDemand/electric.js';
import {usePullEmergency} from '@/traits/emergency/emergency.js';
import {usePullEnergyLevel} from '@/traits/energyStorage/energyStorage.js';
import {usePullEnterLeaveEvents} from '@/traits/enterLeave/enterLeave.js';
import {usePullFanSpeed} from '@/traits/fanSpeed/fanSpeed.js';
import {usePollBrightness, usePullBrightness} from '@/traits/light/light.js';
import {usePullMeterReading} from '@/traits/meter/meter.js';
import {usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {usePullOpenClose} from '@/traits/openClose/openClose.js';
import {usePullCurrentStatus} from '@/traits/status/status.js';
import {computed} from 'vue';

/**
 * The equivalent of using `usePullAirTemperature` (or other traits) but using a named trait.
 *
 * @param {string} trait
 * @param {MaybeRefOrGetter<Object>} request
 * @return {ToRefs<ResourceValue>}
 */
export function usePullTrait(trait, request) {
  const fn = pullTraitByType[trait];
  if (!fn) {
    return {
      streamError: computed(() => 'Trait not supported by usePullTrait: ' + trait)
    };
  }
  return fn(request);
}

/**
 * An object that maps traitName[:traitResource] to the composable function that pulls data for that trait.
 * Keys are the fully qualified trait names optionally followed by the resource name to fetch.
 * If no resource name is specified the default resource will be used.
 *
 * For example `smartcore.traits.AirQualitySensor` will act the same as `smartcore.traits.AirQualitySensor:AirQuality`.
 * Resource names use the singular form, that of the message type returned when getting that type of resource.
 *
 * @type {{[key: string]: function(MaybeRefOrGetter<string|Object>): ToRefs<ResourceValue<*, *>>}}
 */
export const pullTraitByType = {
  'smartcore.bos.Access': usePullAccessAttempts,
  'smartcore.bos.Access:AccessAttempt': usePullAccessAttempts,
  'smartcore.traits.AirQualitySensor': usePullAirQuality,
  'smartcore.traits.AirQualitySensor:AirQuality': usePullAirQuality,
  'smartcore.traits.AirTemperature': usePullAirTemperature,
  'smartcore.traits.AirTemperature:AirTemperature': usePullAirTemperature,
  'smartcore.traits.Electric': usePullElectricDemand,
  'smartcore.traits.Electric:ElectricDemand': usePullElectricDemand,
  'smartcore.traits.Emergency': usePullEmergency,
  'smartcore.traits.Emergency:Emergency': usePullEmergency,
  'smartcore.traits.EnergyStorage': usePullEnergyLevel,
  'smartcore.traits.EnergyStorage:EnergyLevel': usePullEnergyLevel,
  'smartcore.traits.EnterLeaveSensor': usePullEnterLeaveEvents,
  'smartcore.traits.EnterLeaveSensor:EnterLeaveEvent': usePullEnterLeaveEvents,
  'smartcore.traits.FanSpeed': usePullFanSpeed,
  'smartcore.traits.FanSpeed:FanSpeed': usePullFanSpeed,
  'smartcore.traits.Light': usePullBrightness,
  'smartcore.traits.Light:Brightness': usePullBrightness,
  'smartcore.bos.Meter': usePullMeterReading,
  'smartcore.bos.Meter:Reading': usePullMeterReading,
  'smartcore.traits.OccupancySensor': usePullOccupancy,
  'smartcore.traits.OccupancySensor:Occupancy': usePullOccupancy,
  'smartcore.traits.OpenClose': usePullOpenClose,
  'smartcore.traits.OpenClose:Position': usePullOpenClose,
  'smartcore.bos.Status': usePullCurrentStatus,
  'smartcore.bos.Status:CurrentStatus': usePullCurrentStatus
};

/**
 * The equivalent of periodically calling `usePollAirTemperature` (or other traits) but using a named trait.
 *
 * @param {string} trait - The fully qualified trait name (e.g., 'smartcore.traits.Light').
 * @param {string} name - The resource name or identifier to poll.
 * @return {ToRefs<ResourceValue>} An object containing reactive references to the polled resource value and error state.
 */
export function usePollTrait(trait, name) {
  const fn = pollTraitByType[trait];
  if (!fn) {
    return {
      streamError: computed(() => 'Trait not supported by usePollTrait: ' + trait)
    };
  }
  
  return fn(name);
}

export const pollTraitByType = {
  'smartcore.traits.Light': usePollBrightness,
}