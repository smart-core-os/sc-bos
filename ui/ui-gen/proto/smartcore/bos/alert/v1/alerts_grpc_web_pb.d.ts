import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_alert_v1_alerts_pb from '../../../../smartcore/bos/alert/v1/alerts_pb'; // proto import: "smartcore/bos/alert/v1/alerts.proto"


export class AlertApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAlerts(
    request: smartcore_bos_alert_v1_alerts_pb.ListAlertsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.ListAlertsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.ListAlertsResponse>;

  pullAlerts(
    request: smartcore_bos_alert_v1_alerts_pb.PullAlertsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.PullAlertsResponse>;

  acknowledgeAlert(
    request: smartcore_bos_alert_v1_alerts_pb.AcknowledgeAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.Alert) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.Alert>;

  unacknowledgeAlert(
    request: smartcore_bos_alert_v1_alerts_pb.AcknowledgeAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.Alert) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.Alert>;

  getAlertMetadata(
    request: smartcore_bos_alert_v1_alerts_pb.GetAlertMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.AlertMetadata) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.AlertMetadata>;

  pullAlertMetadata(
    request: smartcore_bos_alert_v1_alerts_pb.PullAlertMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.PullAlertMetadataResponse>;

}

export class AlertAdminApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createAlert(
    request: smartcore_bos_alert_v1_alerts_pb.CreateAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.Alert) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.Alert>;

  updateAlert(
    request: smartcore_bos_alert_v1_alerts_pb.UpdateAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.Alert) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.Alert>;

  resolveAlert(
    request: smartcore_bos_alert_v1_alerts_pb.ResolveAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.Alert) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.Alert>;

  deleteAlert(
    request: smartcore_bos_alert_v1_alerts_pb.DeleteAlertRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_alert_v1_alerts_pb.DeleteAlertResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.DeleteAlertResponse>;

}

export class AlertApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAlerts(
    request: smartcore_bos_alert_v1_alerts_pb.ListAlertsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.ListAlertsResponse>;

  pullAlerts(
    request: smartcore_bos_alert_v1_alerts_pb.PullAlertsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.PullAlertsResponse>;

  acknowledgeAlert(
    request: smartcore_bos_alert_v1_alerts_pb.AcknowledgeAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.Alert>;

  unacknowledgeAlert(
    request: smartcore_bos_alert_v1_alerts_pb.AcknowledgeAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.Alert>;

  getAlertMetadata(
    request: smartcore_bos_alert_v1_alerts_pb.GetAlertMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.AlertMetadata>;

  pullAlertMetadata(
    request: smartcore_bos_alert_v1_alerts_pb.PullAlertMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_alert_v1_alerts_pb.PullAlertMetadataResponse>;

}

export class AlertAdminApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createAlert(
    request: smartcore_bos_alert_v1_alerts_pb.CreateAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.Alert>;

  updateAlert(
    request: smartcore_bos_alert_v1_alerts_pb.UpdateAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.Alert>;

  resolveAlert(
    request: smartcore_bos_alert_v1_alerts_pb.ResolveAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.Alert>;

  deleteAlert(
    request: smartcore_bos_alert_v1_alerts_pb.DeleteAlertRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_alert_v1_alerts_pb.DeleteAlertResponse>;

}

