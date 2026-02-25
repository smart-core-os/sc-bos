import * as grpcWeb from "grpc-web";
import {Message} from "google-protobuf";
import {Timestamp} from "google-protobuf/google/protobuf/timestamp_pb";
import {ChangeType} from "@smart-core-os/sc-api-grpc-web/types/change_pb";

type Opt<T> = T | null | undefined;
type Msg<T> = Message & { toObject(includeInstance?: boolean): T };

export function closeResource(resource: RemoteResource<any> | null);

export function setValue<V>(resource: ResourceValue<V, any>, val: V);

export function setCollection<V, M extends Msg<V>>(resource: ResourceCollection<V, M>, change: CollectionChange<V, M>, idFunc: (T) => string)

export function setError(resource: RemoteResource<any>, err: Error);

export function pullResource<M, V>(logPrefix: string, resource: RemoteResource<M>, newStream: StreamFactory<M>);

export function trackAction<V, M>(logPrefix: string, tracker: ActionTracker<V>, action: Action<V, M>): Promise<V>

export function newActionTracker<T>(): ActionTracker<T>;

export function newResourceValue<T, M extends Msg<T>>(): ResourceValue<T, M>;

export function newResourceCollection<T, M extends Msg<T>>(): ResourceCollection<T, M>;

export interface ResourceError {
  name: string;
  error: grpcWeb.RpcError;
}

export interface RemoteResource<M> {
  loading?: boolean;
  stream?: grpcWeb.ClientReadableStream<M>;
  streamError?: ResourceError;
  updateTime?: Date;
}

export interface ResourceValue<V, M extends Msg<?>> extends RemoteResource<M> {
  value?: V;
}

export interface ResourceCollection<V, M extends Msg<?>> extends RemoteResource<any> {
  value?: { [id: string]: V };
  lastResponse?: M;
}

export interface ResourceCallback<V> {
  data(val: V);

  error(e: Error);
}

export type StreamFactory<M> = (endpoint: string) => grpcWeb.ClientReadableStream<M>;

export interface CollectionChange<V, M extends Msg<V>> {
  getName(): string;

  getChangeTime(): Timestamp | undefined;

  getChangeType(): ChangeType;

  getOldValue(): M | undefined;

  getNewValue(): M | undefined;
}

export interface ActionTracker<V> {
  loading?: boolean;
  error?: ResourceError;
  response?: V;
  duration?: number;
}

export type Action<V, M extends Msg<V>> = (endpoint: string) => Promise<M>;
