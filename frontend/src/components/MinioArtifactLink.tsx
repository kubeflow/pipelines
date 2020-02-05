import * as React from 'react';
import { StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { generateArtifactUrl } from '../lib/Utils';
import { Apis } from '../lib/Apis';

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const MinioArtifactLink: React.FC<S3Artifact> = s3artifact => {
  const { key, bucket, endpoint } = s3artifact || {};
  if (!key || !bucket) {
    return null;
  }

  const peek = 100;
  const encodedKey = encodeURIComponent(key);
  const source = endpoint === 's3.amazonaws.com' ? StorageService.S3 : StorageService.MINIO;
  const linkText = `${source.toString()}://${bucket}/${encodedKey}`;
  const url = generateArtifactUrl(source, bucket, encodedKey);
  const [content, setContent] = React.useState(linkText);

  React.useEffect(() => {
    Apis.readFile({ source, bucket, key }, peek).then(
      data => setContent(data.length > peek ? `${data} ...` : data),
      error => setContent(error),
    );
  }, [source, bucket, key]);

  // Opens in new window safely
  return (
    <a href={url} target={'_blank'} rel={'noreferrer noopener'} title={linkText}>
      {content}
    </a>
  );
};

export default MinioArtifactLink;
