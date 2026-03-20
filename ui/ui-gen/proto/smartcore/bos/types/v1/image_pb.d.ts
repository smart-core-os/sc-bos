import * as jspb from 'google-protobuf'



export class Image extends jspb.Message {
  getSourcesList(): Array<Image.Source>;
  setSourcesList(value: Array<Image.Source>): Image;
  clearSourcesList(): Image;
  addSources(value?: Image.Source, index?: number): Image.Source;

  getDescription(): string;
  setDescription(value: string): Image;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Image.AsObject;
  static toObject(includeInstance: boolean, msg: Image): Image.AsObject;
  static serializeBinaryToWriter(message: Image, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Image;
  static deserializeBinaryFromReader(message: Image, reader: jspb.BinaryReader): Image;
}

export namespace Image {
  export type AsObject = {
    sourcesList: Array<Image.Source.AsObject>;
    description: string;
  };

  export class Content extends jspb.Message {
    getType(): string;
    setType(value: string): Content;

    getBody(): Uint8Array | string;
    getBody_asU8(): Uint8Array;
    getBody_asB64(): string;
    setBody(value: Uint8Array | string): Content;
    hasBody(): boolean;
    clearBody(): Content;

    getUrl(): string;
    setUrl(value: string): Content;
    hasUrl(): boolean;
    clearUrl(): Content;

    getRef(): string;
    setRef(value: string): Content;
    hasRef(): boolean;
    clearRef(): Content;

    getPath(): string;
    setPath(value: string): Content;
    hasPath(): boolean;
    clearPath(): Content;

    getContentCase(): Content.ContentCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Content.AsObject;
    static toObject(includeInstance: boolean, msg: Content): Content.AsObject;
    static serializeBinaryToWriter(message: Content, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Content;
    static deserializeBinaryFromReader(message: Content, reader: jspb.BinaryReader): Content;
  }

  export namespace Content {
    export type AsObject = {
      type: string;
      body?: Uint8Array | string;
      url?: string;
      ref?: string;
      path?: string;
    };

    export enum ContentCase {
      CONTENT_NOT_SET = 0,
      BODY = 2,
      URL = 3,
      REF = 4,
      PATH = 5,
    }
  }


  export class Source extends jspb.Message {
    getSrcList(): Array<Image.Content>;
    setSrcList(value: Array<Image.Content>): Source;
    clearSrcList(): Source;
    addSrc(value?: Image.Content, index?: number): Image.Content;

    getWidth(): number;
    setWidth(value: number): Source;

    getHeight(): number;
    setHeight(value: number): Source;

    getPurpose(): Image.Source.Purpose;
    setPurpose(value: Image.Source.Purpose): Source;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Source.AsObject;
    static toObject(includeInstance: boolean, msg: Source): Source.AsObject;
    static serializeBinaryToWriter(message: Source, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Source;
    static deserializeBinaryFromReader(message: Source, reader: jspb.BinaryReader): Source;
  }

  export namespace Source {
    export type AsObject = {
      srcList: Array<Image.Content.AsObject>;
      width: number;
      height: number;
      purpose: Image.Source.Purpose;
    };

    export enum Purpose {
      PURPOSE_UNSPECIFIED = 0,
      ANY = 1,
      MASKABLE = 2,
    }
  }

}

