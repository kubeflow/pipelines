import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';
import { StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { buildQuery } from '../lib/Utils';
import { isS3Endpoint } from '../lib/AwsHelper';
import { Apis } from '../lib/Apis';

const css = stylesheet({
  preview: {
    maxHeight: 250,
    overflowY: 'auto',
    padding: 3,
    backgroundColor: color.lightGrey,
  },
});

/**
 * Check if a javascript object is an argo S3Artifact object.
 * @param value Any javascript object.
 */
export function isS3Artifact(value: any): value is S3Artifact {
  return value && value.key && value.bucket;
}

export interface MinioArtifactPreviewProps {
  artifact?: S3Artifact;
  namespace?: string;
  peek?: number;
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const MinioArtifactPreview: React.FC<MinioArtifactPreviewProps> = ({
  artifact,
  namespace,
  peek = 255,
}) => {
  const [content, setContent] = React.useState<string | undefined>(undefined);
  const { key, bucket, endpoint } = artifact || {};
  const source = isS3Endpoint(endpoint) ? StorageService.S3 : StorageService.MINIO;

  React.useEffect(() => {
    let cancelled = false;
    if (bucket && key) {
      Apis.readFile({ source, bucket, key }, namespace, peek + 1).then(
        data => !cancelled && setContent(data.length > peek ? `${data.slice(0, peek)} ...` : data),
        error => console.error(error),
      );
    }
    return () => {
      cancelled = true;
    };
  }, [source, bucket, key, peek, namespace]);

  if (!key || !bucket) {
    return null;
  }

  // TODO need to come to an agreement how to encode artifact info inside a url
  // namespace is currently not supported
  const linkText = `${source.toString()}://${bucket}/${key}`;
  // TODO better way to generate url? Apis.readFile executes the query only.
  const url = `artifacts/get${buildQuery({ source, bucket, key, namespace })}`;

  // Opens in new window safely
  // TODO use ArtifactLink instead (but it need to support namespace)
  return (
    <div>
      <a href={url} target='_blank' rel='noreferrer noopener' title={linkText}>
        {linkText}
      </a>
      {content && (
        <div className={css.preview}>
          <small>
            <pre>{content}</pre>
          </small>
        </div>
      )}
    </div>
  );
};

export default MinioArtifactPreview;
