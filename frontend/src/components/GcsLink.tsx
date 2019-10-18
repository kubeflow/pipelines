import * as React from 'react';
import { generateGcsConsoleUri } from '../lib/Utils';

/**
 * A component that renders a gcs console link when gcsUri is gs:// and pure
 * text if it is not a valid gs:// uri.
 */
export const GcsLink: React.FC<{ gcsUri?: string }> = ({ gcsUri }) => {
  const gcsConsoleUri = gcsUri ? generateGcsConsoleUri(gcsUri) : undefined;
  if (gcsConsoleUri) {
    // Opens in new window safely
    return (
      <a href={gcsConsoleUri} target={'_blank'} rel={'noreferrer noopener'}>
        {gcsUri}
      </a>
    );
  } else {
    return <>{gcsUri}</>;
  }
};
