import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';
import { StorageService, StoragePath } from '../lib/WorkflowParser';
import { S3Artifact } from '../../third_party/argo-ui/argo_template';
import { isS3Endpoint } from '../lib/AwsHelper';
import { Apis } from '../lib/Apis';
import { ExternalLink } from '../atoms/ExternalLink';
import { ValueComponentProps } from './DetailsTable';

const css = stylesheet({
  root: {
    width: '100%',
  },
  preview: {
    backgroundColor: color.lightGrey,
    flex: 1,
    maxHeight: 250,
    overflowY: 'auto',
    padding: 3,
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
  maxbytes?: number;
  maxlines?: number;
}

function getStoragePath(value?: string | Partial<S3Artifact>) {
  if (!value || typeof value === 'string') return;
  const { key, bucket, endpoint } = value;
  if (!bucket || !key) return;
  const source = isS3Endpoint(endpoint) ? StorageService.S3 : StorageService.MINIO;
  return { source, bucket, key };
}

async function getPreview(
  storagePath: StoragePath,
  namespace: string | undefined,
  maxbytes: number,
  maxlines?: number,
): Promise<{ data: string; hasMore: boolean }> {
  // TODO how to handle binary data (can probably use magic number to id common mime types)
  let data = await Apis.readFile(storagePath, namespace, maxbytes + 1);
  // is preview === data and no maxlines
  if (data.length <= maxbytes && !maxlines) {
    return { data, hasMore: false };
  }
  // remove extra byte at the end (we requested maxbytes +1)
  data = data.slice(0, maxbytes);
  // check num lines
  if (maxlines) {
    data = data
      .split('\n')
      .slice(0, maxlines)
      .join('\n')
      .trim();
  }
  return { data: `${data}\n...`, hasMore: true };
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const MinioArtifactPreview: React.FC<MinioArtifactPreviewProps> = ({
  value,
  namespace,
  maxbytes = 255,
  maxlines,
}) => {
  const [content, setContent] = React.useState<{ data: string; hasMore: boolean } | undefined>(
    undefined,
  );
  const storagePath = getStoragePath(value);

  React.useEffect(() => {
    let cancelled = false;
    if (storagePath) {
      getPreview(storagePath, namespace, maxbytes, maxlines).then(
        content => !cancelled && setContent(content),
        error => console.error(error), // TODO error badge on link?
      );
    }
    return () => {
      cancelled = true;
    };
  }, [storagePath?.source, storagePath?.bucket, storagePath?.key, namespace, maxbytes, maxlines]);

  if (!storagePath) {
    // if value is undefined, null, or an invalid s3artifact object, don't render
    if (value === null || value === undefined || typeof value === 'object') return null;
    // otherwise render value as string (with default string method)
    return <React.Fragment>{`${value}`}</React.Fragment>;
  }

  // TODO need to come to an agreement how to encode artifact info inside a url
  // namespace is currently not supported
  const linkText = Apis.buildArtifactUrl(storagePath);
  const artifactDownloadUrl = Apis.buildReadFileUrl({
    path: storagePath,
    namespace,
    isDownload: true,
  });
  const artifactViewUrl = Apis.buildReadFileUrl({ path: storagePath, namespace });

  // Opens in new window safely
  // TODO use ArtifactLink instead (but it need to support namespace)
  return (
    <div className={css.root}>
      <ExternalLink href={artifactDownloadUrl} title={linkText}>
        {linkText}
      </ExternalLink>
      {content?.data && (
        <div className={css.preview}>
          <small>
            <pre>{content.data}</pre>
          </small>
          {content.hasMore && <ExternalLink href={artifactViewUrl}>View All</ExternalLink>}
        </div>
      )}
    </div>
  );
};

export default MinioArtifactPreview;
