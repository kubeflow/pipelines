import * as React from 'react';
import { StoragePath, StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';

const artifactApiUri = ({ source, bucket, key }: StoragePath) =>
  'artifacts/get' +
  `?source=${source}&bucket=${bucket}&key=${encodeURIComponent(key)}`;

/**
 * A component that renders an artifact link.
 */
const MinioArtifactLink: React.FC<S3Artifact> = (
  s3artifact
) => {
  if (!s3artifact || !s3artifact.key || !s3artifact.bucket) {
    return null;
  }

  const { key, bucket, endpoint } = s3artifact;
  const source = endpoint === 's3.amazonaws.com' ? StorageService.S3 : StorageService.MINIO;
  const linkText = `${source.toString()}://${bucket}/${key}`;
  // Opens in new window safely
  return (
    <a href={artifactApiUri({key, bucket, source})} target={'_blank'} rel={'noreferrer noopener'} title={linkText}>
      {linkText}
    </a>
  );

};

export default MinioArtifactLink;