/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useContext, useEffect, useState } from 'react';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { GkeMetadataContext } from 'src/lib/GkeMetadata';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { Deployments, KFP_FLAGS } from 'src/lib/Flags';
import { AppShell } from './AppShell';
import { CommandPalette } from '../navigation/CommandPalette';
import type { AppShellProps } from './AppShell';
import { ThemeProvider } from './ThemeProvider';
import './ApplicationShell.css';

type ApplicationShellProps = Pick<
  AppShellProps,
  'children' | 'items' | 'currentPath' | 'secondaryItems'
>;

export function ApplicationShell(props: ApplicationShellProps) {
  const [searchOpen, setSearchOpen] = useState(false);
  // External sync: one document shortcut shared by the normal and embedded shell.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        (event.ctrlKey || event.metaKey) &&
        event.key.toLowerCase() === 'k' &&
        !event.altKey &&
        !event.repeat &&
        !event.isComposing
      ) {
        event.preventDefault();
        setSearchOpen(true);
      }
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, []);
  const buildInfo = useContext(BuildInfoContext);
  const gkeMetadata = useContext(GkeMetadataContext);
  const namespace = useContext(NamespaceContext);
  const commitHash = buildInfo?.apiServerCommitHash || buildInfo?.frontendCommitHash || '';
  const version = buildInfo?.apiServerTagName || buildInfo?.frontendTagName || 'unknown';
  const versionHref =
    'https://github.com/kubeflow/pipelines' +
    (commitHash && commitHash !== 'unknown' ? `/commit/${commitHash}` : '');
  const clusterQuery = new URLSearchParams({
    project: gkeMetadata.projectId || '',
    filter: `name:${gkeMetadata.clusterName || ''}`,
  });
  const clusterHref =
    gkeMetadata.projectId && gkeMetadata.clusterName
      ? `https://console.cloud.google.com/kubernetes/list?${clusterQuery}`
      : undefined;

  return (
    <ThemeProvider className='kfp-application-theme'>
      <AppShell
        {...props}
        hideSideNav={KFP_FLAGS.HIDE_SIDENAV}
        namespace={namespace}
        onSearch={() => setSearchOpen(true)}
        version={version}
        versionHref={versionHref}
        metadata={{
          clusterName: clusterHref ? gkeMetadata.clusterName : undefined,
          clusterHref,
          projectId: clusterHref ? gkeMetadata.projectId : undefined,
          buildDate: buildInfo?.buildDate
            ? new Date(buildInfo.buildDate).toLocaleDateString('en-US')
            : 'unknown',
          commitHash: commitHash ? commitHash.substring(0, 7) : 'unknown',
        }}
      />
      <CommandPalette
        open={searchOpen}
        onClose={() => setSearchOpen(false)}
        namespace={namespace}
        requireNamespace={
          buildInfo?.apiServerMultiUser || KFP_FLAGS.DEPLOYMENT === Deployments.KUBEFLOW
        }
        items={[...props.items, ...(props.secondaryItems || [])]}
      />
    </ThemeProvider>
  );
}
