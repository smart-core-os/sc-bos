import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_securityevent_v1_security_event_pb from '../../../../smartcore/bos/securityevent/v1/security_event_pb'; // proto import: "smartcore/bos/securityevent/v1/security_event.proto"


export class SecurityEventApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSecurityEvents(
    request: smartcore_bos_securityevent_v1_security_event_pb.ListSecurityEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_securityevent_v1_security_event_pb.ListSecurityEventsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_securityevent_v1_security_event_pb.ListSecurityEventsResponse>;

  pullSecurityEvents(
    request: smartcore_bos_securityevent_v1_security_event_pb.PullSecurityEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_securityevent_v1_security_event_pb.PullSecurityEventsResponse>;

}

export class SecurityEventApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSecurityEvents(
    request: smartcore_bos_securityevent_v1_security_event_pb.ListSecurityEventsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_securityevent_v1_security_event_pb.ListSecurityEventsResponse>;

  pullSecurityEvents(
    request: smartcore_bos_securityevent_v1_security_event_pb.PullSecurityEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_securityevent_v1_security_event_pb.PullSecurityEventsResponse>;

}

