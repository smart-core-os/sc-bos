import * as jspb from 'google-protobuf'



export class PageToken extends jspb.Message {
  getRecordId(): string;
  setRecordId(value: string): PageToken;

  getTotalSize(): number;
  setTotalSize(value: number): PageToken;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PageToken.AsObject;
  static toObject(includeInstance: boolean, msg: PageToken): PageToken.AsObject;
  static serializeBinaryToWriter(message: PageToken, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PageToken;
  static deserializeBinaryFromReader(message: PageToken, reader: jspb.BinaryReader): PageToken;
}

export namespace PageToken {
  export type AsObject = {
    recordId: string;
    totalSize: number;
  };
}

