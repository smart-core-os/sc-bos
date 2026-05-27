import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource.js';
import {MockDeviceApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/mock/v1/mock_grpc_web_pb';
import {
  ForceTraitValuesRequest,
  SetDeviceAutomationsRequest,
  TraitAutomation,
  TraitValue
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/mock/v1/mock_pb';

/**
 * @param {{name: string, values: Array<{trait: string, valueJson: string}>}} request
 * @param {ActionTracker} [tracker]
 * @return {Promise<{}>}
 */
export function forceTraitValue({name, values}, tracker) {
  return trackAction('MockDeviceApi.forceTraitValue', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const dst = new ForceTraitValuesRequest();
    dst.setName(name);
    dst.setValuesList((values ?? []).map(v => {
      const tv = new TraitValue();
      tv.setTrait(v.trait);
      tv.setValueJson(v.valueJson);
      return tv;
    }));
    return api.forceTraitValue(dst);
  });
}

/**
 * @param {{name: string, automations: Array<{trait?: string, id?: string, active: boolean}>}} request
 * @param {ActionTracker} [tracker]
 * @return {Promise<{}>}
 */
export function setDeviceAutomation({name, automations}, tracker) {
  return trackAction('MockDeviceApi.setDeviceAutomation', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const dst = new SetDeviceAutomationsRequest();
    dst.setName(name);
    dst.setAutomationsList((automations ?? []).map(a => {
      const ta = new TraitAutomation();
      ta.setTrait(a.trait ?? '');
      ta.setId(a.id ?? '');
      ta.setActive(a.active);
      return ta;
    }));
    return api.setDeviceAutomation(dst);
  });
}

/**
 * @param {string} endpoint
 * @return {MockDeviceApiPromiseClient}
 */
function apiClient(endpoint) {
  return new MockDeviceApiPromiseClient(endpoint, null, clientOptions());
}
