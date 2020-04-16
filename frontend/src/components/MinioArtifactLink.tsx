import * as React from 'react';
import { StoragePath, StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { buildQuery } from 'src/lib/Utils';

const artifactApiUri = ({ source, bucket, key }: StoragePath, namespace?: string) =>
  `namespaces/${namespace}/artifacts/get${buildQuery({ source, bucket, key })}`;

/**
 * A component that renders an artifact link.
 */
const MinioArtifactLink: React.FC<{
  artifact: Partial<S3Artifact> | null | undefined;
  namespace?: string;
}> = ({ artifact, namespace }) => {
  if (!artifact || !artifact.key || !artifact.bucket) {
    return null;
  }

  const { key, bucket, endpoint } = artifact;
  const source = endpoint === 's3.amazonaws.com' ? StorageService.S3 : StorageService.MINIO;
  const linkText = `${source.toString()}://${bucket}/${key}`;
  // Opens in new window safely
  return (
    <a
      href={artifactApiUri({ key, bucket, source }, namespace)}
      target={'_blank'}
      rel='noreferrer noopener'
      title={linkText}
    >
      {linkText}
    </a>
  );
};

export default MinioArtifactLink;
