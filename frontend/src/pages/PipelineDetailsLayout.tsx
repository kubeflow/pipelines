/*
 * Copyright 2021 The Kubeflow Authors
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
import { InspectionTabs, type InspectionTabsProps } from 'src/components/inspection/InspectionTabs';
import './DetailLayouts.css';
import { PageContainer } from 'src/components/layout/PageContainer';

/** Own the page gutter for every pipeline detail tab, including version management. */
export function PipelineDetailsLayout(props: InspectionTabsProps) {
  return (
    <PageContainer className='kfp-detail-page kfp-pipeline-detail' data-testid='pipeline-detail-v2'>
      <InspectionTabs {...props} />
    </PageContainer>
  );
}
