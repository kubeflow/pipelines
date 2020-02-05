import * as React from 'react';
import { StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { generateArtifactUrl, isS3Endpoint } from '../lib/Utils';
import { Apis } from '../lib/Apis';


/**
 * Check if a javascript object is an argo S3Artifact object.
 * @param value Any javascript object.
 */
export function isS3Artifact(value: any): value is S3Artifact {
  return value && value.key && value.bucket && value.endpoint;
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const MinioArtifactPreview: React.FC<{artifact: S3Artifact}> = ({artifact = {}}) => {
  const { key, bucket, endpoint } = artifact;
  if (!key || !bucket) {
    return null;
  }

  const peek = 100;
  const encodedKey = encodeURIComponent(key);
  const source = isS3Endpoint(endpoint) ? StorageService.S3 : StorageService.MINIO;
  const linkText = `${source.toString()}://${bucket}/${encodedKey}`;
  const url = generateArtifactUrl(source, bucket, encodedKey);
  const [content, setContent] = React.useState('');

  React.useEffect(() => {
    let cancelled = false;
    Apis.readFile({ source, bucket, key }, peek + 1).then(
      data => !cancelled && setContent(data.length > peek ? `${data.slice(0, peek)} ...` : data),
      error => console.error(error),
    );
    return () => {
      cancelled = true;
    };
  }, [source, bucket, key]);

  // Opens in new window safely
  return (
    <div>
      <a href={url} target={'_blank'} rel={'noreferrer noopener'} title={linkText}>
        {linkText}
      </a>
      <div>
        <small><pre>{content}</pre></small>
      </div>
    </div>
  );
};

export default MinioArtifactPreview;
