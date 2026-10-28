// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { InspectionFields } from 'src/components/inspection/InspectionFields';
import { Description } from 'src/components/Description';

export interface PipelineDefinitionSource {
  name: string;
  href: string;
  inline: boolean;
}

const renderValue = ({ value }: { value?: ReactNode }) => <>{value}</>;

export function PipelineDefinitionDetails({
  spec,
  source,
}: {
  spec?: PipelineSpec;
  source?: PipelineDefinitionSource;
}) {
  const parameters = Object.entries(spec?.root?.inputDefinitions?.parameters || {});
  return (
    <div className='kfp-detail-sections'>
      <InspectionFields<ReactNode>
        title='Pipeline definition'
        valueComponent={renderValue}
        fields={[
          ['Name', spec?.pipelineInfo?.name || 'Not specified'],
          [
            'Description',
            <Description description={spec?.pipelineInfo?.description || 'No description'} />,
          ],
          [
            'Source',
            source ? (
              <>
                <span>
                  {source.inline ? 'Inline definition' : 'Referenced pipeline definition'}
                </span>
                {' · '}
                <Link to={source.href}>{source.name}</Link>
              </>
            ) : (
              'Pipeline specification'
            ),
          ],
        ]}
      />
      {parameters.length ? (
        <InspectionFields
          title='Input parameter defaults'
          fields={parameters.map(([name, parameter]) => [
            name,
            parameter.defaultValue === undefined
              ? 'No default'
              : JSON.stringify(parameter.defaultValue),
          ])}
        />
      ) : (
        <p>No input parameters defined.</p>
      )}
    </div>
  );
}
