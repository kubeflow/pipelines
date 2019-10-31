import * as React from 'react';
import { generateGcsConsoleUri } from '../lib/Utils';

/**
 * A component that renders a gcs console link when gcsUri is gs:// and pure
 * text if it is not a valid gs:// uri.
 */
export const GcsLink: React.FC<{ gcsUri?: string }> = ({ gcsUri }) => {
  var clickableUrl: string | undefined = undefined
  if (gcsUri) {
    if (gcsUri.startsWith('gs:')) {
      var gcsConsoleUrl = generateGcsConsoleUri(gcsUri)
      if (gcsConsoleUrl) {
        clickableUrl = gcsConsoleUrl
      }
    } else if (gcsUri.startsWith('http:') || gcsUri.startsWith('https:')) {
      clickableUrl = gcsUri
    }
  }

  if (clickableUrl) {
    // Opens in new window safely
    return (
      <a href={clickableUrl} target={'_blank'} rel={'noreferrer noopener'}>
        {gcsUri}
      </a>
    );
  } else {
    return <>{gcsUri}</>;
  }
};
