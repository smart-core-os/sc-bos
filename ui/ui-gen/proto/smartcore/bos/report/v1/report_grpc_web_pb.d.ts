import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_report_v1_report_pb from '../../../../smartcore/bos/report/v1/report_pb'; // proto import: "smartcore/bos/report/v1/report.proto"


export class ReportApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listReports(
    request: smartcore_bos_report_v1_report_pb.ListReportsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_report_v1_report_pb.ListReportsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_report_v1_report_pb.ListReportsResponse>;

  getDownloadReportUrl(
    request: smartcore_bos_report_v1_report_pb.GetDownloadReportUrlRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_report_v1_report_pb.DownloadReportUrl) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_report_v1_report_pb.DownloadReportUrl>;

}

export class ReportApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listReports(
    request: smartcore_bos_report_v1_report_pb.ListReportsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_report_v1_report_pb.ListReportsResponse>;

  getDownloadReportUrl(
    request: smartcore_bos_report_v1_report_pb.GetDownloadReportUrlRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_report_v1_report_pb.DownloadReportUrl>;

}

