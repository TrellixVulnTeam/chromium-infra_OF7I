// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { Empty } from './common/empty';
import { FieldMask } from './common/field_mask';
import { rpcClient } from './common/rpc_client';
import { fromJsonTimestamp, isSet } from './common/utils';

/** Performs operations on Assets. */
export interface IAssetService {
  /** Creates the given Asset. */
  create(request: CreateAssetRequest): Promise<AssetEntity>;
  /** Retrieves a Asset for a given unique value. */
  get(request: GetAssetRequest): Promise<AssetEntity>;
  /** Update a single asset in poros. */
  update(request: UpdateAssetRequest): Promise<AssetEntity>;
  /** Deletes the given Asset. */
  delete(request: DeleteAssetRequest): Promise<Empty>;
  /** Lists all Assets. */
  list(request: ListAssetsRequest): Promise<ListAssetsResponse>;
}

export class AssetService implements IAssetService {
  constructor() {
    this.create = this.create.bind(this);
    this.get = this.get.bind(this);
    this.update = this.update.bind(this);
    this.delete = this.delete.bind(this);
    this.list = this.list.bind(this);
  }

  create = (request: CreateAssetRequest): Promise<AssetEntity> => {
    const data = CreateAssetRequest.toJSON(request);
    const promise = rpcClient.request('poros.Asset', 'Create', data);
    return promise.then((data) => AssetEntity.fromJSON(data));
  };

  get = (request: GetAssetRequest): Promise<AssetEntity> => {
    const data = GetAssetRequest.toJSON(request);
    const promise = rpcClient.request('poros.Asset', 'Get', data);
    return promise.then((data) => AssetEntity.fromJSON(data));
  };

  update = (request: UpdateAssetRequest): Promise<AssetEntity> => {
    const data = UpdateAssetRequest.toJSON(request);
    const promise = rpcClient.request('poros.Asset', 'Update', data);
    return promise.then((data) => AssetEntity.fromJSON(data));
  };

  delete = (request: DeleteAssetRequest): Promise<Empty> => {
    const data = DeleteAssetRequest.toJSON(request);
    const promise = rpcClient.request('poros.Asset', 'Delete', data);
    return promise.then((data) => Empty.fromJSON(data));
  };

  list = (request: ListAssetsRequest): Promise<ListAssetsResponse> => {
    const data = ListAssetsRequest.toJSON(request);
    const promise = rpcClient.request('poros.Asset', 'List', data);
    return promise.then((data) => ListAssetsResponse.fromJSON(data));
  };
}

export interface AssetEntity {
  /** Unique identifier of the asset */
  assetId: string;
  /** Name of the asset */
  name: string;
  /** Description of the asset */
  description: string;
  /** User who created the record */
  createdBy: string;
  /** Timestamp for the creation of the record */
  createdAt: Date | undefined;
  /** User who modified the record */
  modifiedBy: string;
  /** Timestamp for the last update of the record */
  modifiedAt: Date | undefined;
}

export const AssetEntity = {
  defaultEntity(): AssetEntity {
    return {
      assetId: '',
      name: '',
      description: '',
      createdBy: '',
      createdAt: undefined,
      modifiedBy: '',
      modifiedAt: undefined,
    };
  },
  fromJSON(object: any): AssetEntity {
    console.log(object);
    return {
      assetId: isSet(object.assetId) ? String(object.assetId) : '',
      name: isSet(object.name) ? String(object.name) : '',
      description: isSet(object.description) ? String(object.description) : '',
      createdBy: isSet(object.createdBy) ? String(object.createdBy) : '',
      createdAt: isSet(object.createdAt)
        ? fromJsonTimestamp(object.createdAt)
        : undefined,
      modifiedBy: isSet(object.modifiedBy) ? String(object.modifiedBy) : '',
      modifiedAt: isSet(object.modifiedAt)
        ? fromJsonTimestamp(object.modifiedAt)
        : undefined,
    };
  },

  toJSON(message: AssetEntity): unknown {
    const obj: any = {};
    message.assetId !== undefined && (obj.assetId = message.assetId);
    message.name !== undefined && (obj.name = message.name);
    message.description !== undefined &&
      (obj.description = message.description);
    message.createdBy !== undefined && (obj.createdBy = message.createdBy);
    message.createdAt !== undefined &&
      (obj.createdAt = message.createdAt.toISOString());
    message.modifiedBy !== undefined && (obj.modifiedBy = message.modifiedBy);
    message.modifiedAt !== undefined &&
      (obj.modifiedAt = message.modifiedAt.toISOString());
    return obj;
  },
};

/** Request to create a single asset in AssetServ */
export interface CreateAssetRequest {
  /** Name of the asset */
  name: string;
  /** Description of the asset */
  description: string;
  /** User who created the record */
}

export const CreateAssetRequest = {
  toJSON(message: CreateAssetRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.description !== undefined &&
      (obj.description = message.description);
    return obj;
  },
};

// Request to delete a single asset from poros.
export interface DeleteAssetRequest {
  /** Unique identifier of the asset */
  id: string;
}

export const DeleteAssetRequest = {
  toJSON(message: DeleteAssetRequest): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.name = message.id);
    return obj;
  },
};

/** Gets a Asset resource. */
export interface GetAssetRequest {
  /**
   * The name of the asset to retrieve.
   * Format: publishers/{publisher}/assets/{asset}
   */
  id: string;
}

export const GetAssetRequest = {
  toJSON(message: GetAssetRequest): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    return obj;
  },
};

/** Request to list all assets in poros. */
export interface ListAssetsRequest {
  /** Fields to include on each result */
  readMask: string[] | undefined;
  /** Number of results per page */
  pageSize: number;
  /** Page token from a previous page's ListAssetsResponse */
  pageToken: string;
}

/** Response to ListAssetsRequest. */
export interface ListAssetsResponse {
  /** The result set. */
  assets: AssetEntity[];
  /**
   * A page token that can be passed into future requests to get the next page.
   * Empty if there is no next page.
   */
  nextPageToken: string;
}

export const ListAssetsRequest = {
  toJSON(message: ListAssetsRequest): unknown {
    const obj: any = {};
    message.readMask !== undefined &&
      (obj.readMask = FieldMask.toJSON(FieldMask.wrap(message.readMask)));
    message.pageSize !== undefined &&
      (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },
};

export const ListAssetsResponse = {
  fromJSON(object: any): ListAssetsResponse {
    return {
      assets: Array.isArray(object?.assets)
        ? object.assets.map((e: any) => AssetEntity.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken)
        ? String(object.nextPageToken)
        : '',
    };
  },
};

/** Request to update a single asset in poros. */
export interface UpdateAssetRequest {
  /** The existing asset to update in the database. */
  asset: AssetEntity | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export const UpdateAssetRequest = {
  toJSON(message: UpdateAssetRequest): unknown {
    const obj: any = {};
    message.asset !== undefined &&
      (obj.asset = message.asset
        ? AssetEntity.toJSON(message.asset)
        : undefined);
    message.updateMask !== undefined &&
      (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },
};
