import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_mock_v1_mock_pb from '../../../../smartcore/bos/mock/v1/mock_pb'; // proto import: "smartcore/bos/mock/v1/mock.proto"


export class MockDeviceApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  forceTraitValue(
    request: smartcore_bos_mock_v1_mock_pb.ForceTraitValuesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_mock_v1_mock_pb.ForceTraitValuesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_mock_v1_mock_pb.ForceTraitValuesResponse>;

  setDeviceAutomation(
    request: smartcore_bos_mock_v1_mock_pb.SetDeviceAutomationsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_mock_v1_mock_pb.SetDeviceAutomationsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_mock_v1_mock_pb.SetDeviceAutomationsResponse>;

}

export class MockDeviceApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  forceTraitValue(
    request: smartcore_bos_mock_v1_mock_pb.ForceTraitValuesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_mock_v1_mock_pb.ForceTraitValuesResponse>;

  setDeviceAutomation(
    request: smartcore_bos_mock_v1_mock_pb.SetDeviceAutomationsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_mock_v1_mock_pb.SetDeviceAutomationsResponse>;

}

