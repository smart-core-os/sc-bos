import {setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue} from '@/api/resource';
import {UdmiServicePromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/udmi/v1/udmi_grpc_web_pb';
import {PullExportMessagesRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/udmi/v1/udmi_pb';

/**
 *
 * @param {Partial<PullExportMessagesRequest.AsObject>} request
 * @param {ResourceValue<MqttMessage.AsObject, PullExportMessagesResponse>} resource
 */
export function pullExportMessages(request, resource) {
  pullResource('UDMI.pullExportMessages', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullExportMessages(pullExportMessagesRequestFromObject(request));
    stream.on('data', msg => {
      setValue(resource, msg.getMessage().toObject());
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {UdmiServicePromiseClient}
 */
function apiClient(endpoint) {
  return new UdmiServicePromiseClient(endpoint, null, clientOptions());
}

/**
 *
 * @param {Partial<PullExportMessagesRequest.AsObject>} obj
 * @return {undefined|PullExportMessagesRequest}
 */
function pullExportMessagesRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullExportMessagesRequest();
  setProperties(dst, obj, 'name', 'includeLast');
  return dst;
}
