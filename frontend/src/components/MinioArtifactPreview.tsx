import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';
import { StorageService } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { isS3Endpoint } from '../lib/AwsHelper';
import { Apis } from '../lib/Apis';
import { ExternalLink } from '../atoms/ExternalLink';
import { ValueComponentProps } from './DetailsTable';

const css = stylesheet({
  root: {
    width: '100%',
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
  },
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

export interface MinioArtifactPreviewProps extends ValueComponentProps<Partial<S3Artifact>> {
  namespace?: string;
  peek?: number;
}

function getStoragePath(value?: string | Partial<S3Artifact>) {
  if (!value || typeof value === 'string') return;
  const { key, bucket, endpoint } = value;
  if (!bucket || !key) return;
  const source = isS3Endpoint(endpoint) ? StorageService.S3 : StorageService.MINIO;
  return { source, bucket, key };
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const MinioArtifactPreview: React.FC<MinioArtifactPreviewProps> = ({
  value,
  namespace,
  peek = 255,
}) => {
  const [content, setContent] = React.useState<string | undefined>(undefined);
  const storagePath = getStoragePath(value);

  React.useEffect(() => {
    let cancelled = false;
    if (storagePath) {
      // TODO how to handle binary data
      Apis.readFile(storagePath, namespace, peek + 1).then(
        data => !cancelled && setContent(data.length > peek ? `${data.slice(0, peek)} ...` : data),
        error => console.error(error),
      );
    }
    return () => {
      cancelled = true;
    };
  }, [storagePath, peek, namespace]);

  if (!storagePath) {
    // return value as is if it is defined
    return value ? <React.Fragment>{`${value}`}</React.Fragment> : null;
  }

  // TODO need to come to an agreement how to encode artifact info inside a url
  // namespace is currently not supported
  const linkText = Apis.buildArtifactUrl(storagePath);
  const artifactUrl = Apis.buildReadFileUrl(storagePath, namespace, peek);

  // Opens in new window safely
  // TODO use ArtifactLink instead (but it need to support namespace)
  return (
    <div className={css.root}>
      <ExternalLink href={artifactUrl} title={linkText}>
        {linkText}
      </ExternalLink>
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
