export enum StorageService {
  GCS = 'gcs',
  MINIO = 'minio',
}

export interface StoragePath {
  source: StorageService;
  bucket: string;
  key: string;
}
