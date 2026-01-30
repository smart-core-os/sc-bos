import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_lightingtest_v1_lighting_test_pb from '../../../../smartcore/bos/lightingtest/v1/lighting_test_pb'; // proto import: "smartcore/bos/lightingtest/v1/lighting_test.proto"


export class LightingTestApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLightHealth(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.GetLightHealthRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lightingtest_v1_lighting_test_pb.LightHealth) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lightingtest_v1_lighting_test_pb.LightHealth>;

  listLightHealth(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightHealthRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightHealthResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightHealthResponse>;

  listLightEvents(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightEventsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightEventsResponse>;

  getReportCSV(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.GetReportCSVRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lightingtest_v1_lighting_test_pb.ReportCSV) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lightingtest_v1_lighting_test_pb.ReportCSV>;

}

export class LightingTestApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLightHealth(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.GetLightHealthRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lightingtest_v1_lighting_test_pb.LightHealth>;

  listLightHealth(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightHealthRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightHealthResponse>;

  listLightEvents(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightEventsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lightingtest_v1_lighting_test_pb.ListLightEventsResponse>;

  getReportCSV(
    request: smartcore_bos_lightingtest_v1_lighting_test_pb.GetReportCSVRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lightingtest_v1_lighting_test_pb.ReportCSV>;

}

