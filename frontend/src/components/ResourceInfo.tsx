/*
 * Copyright 2019 The Kubeflow Authors
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
import { Artifact, Execution, getMetadataValue } from '@kubeflow/frontend';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color, commonCss } from '../Css';
import { ArtifactLink } from './ArtifactLink';

export const css = stylesheet({
  field: {
    flexBasis: '300px',
    marginBottom: '32px',
    padding: '4px',
  },
  resourceInfo: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
  },
  term: {
    color: color.grey,
    fontSize: '12px',
    letterSpacing: '0.2px',
    lineHeight: '16px',
  },
  value: {
    color: color.secondaryText,
    fontSize: '14px',
    letterSpacing: '0.2px',
    lineHeight: '20px',
  },
});

export enum ResourceType {
  ARTIFACT = 'ARTIFACT',
  EXECUTION = 'EXECUTION',
}

interface ArtifactProps {
  resourceType: ResourceType.ARTIFACT;
  resource: Artifact;
  typeName: string;
}

interface ExecutionProps {
  resourceType: ResourceType.EXECUTION;
  resource: Execution;
  typeName: string;
}

export type ResourceInfoProps = ArtifactProps | ExecutionProps;

export class ResourceInfo extends React.Component<ResourceInfoProps, {}> {
  public render(): JSX.Element {
    const { resource } = this.props;
    const propertyMap = resource.getPropertiesMap();
    const customPropertyMap = resource.getCustomPropertiesMap();
    return (
      <section>
        <h1 className={commonCss.header}>Type: {this.props.typeName}</h1>
        {(() => {
          if (this.props.resourceType === ResourceType.ARTIFACT) {
            return (
              <>
                <dt className={css.term}>URI</dt>
                <dd className={css.value}>
                  <ArtifactLink artifactUri={this.props.resource.getUri()} />
                </dd>
              </>
            );
          }
          return null;
        })()}
        <h2 className={commonCss.header2}>Properties</h2>
        <dl className={css.resourceInfo}>
          {propertyMap
            .getEntryList()
            // TODO: __ALL_META__ is something of a hack, is redundant, and can be ignored
            .filter(k => k[0] !== '__ALL_META__')
            .map(k => (
              <div className={css.field} key={k[0]} data-testid='resource-info-property'>
                <dt className={css.term} data-testid='resource-info-property-key'>
                  {k[0]}
                </dt>
                <dd className={css.value} data-testid='resource-info-property-value'>
                  {propertyMap && prettyPrintValue(getMetadataValue(propertyMap.get(k[0])))}
                </dd>
              </div>
            ))}
        </dl>
        <h2 className={commonCss.header2}>Custom Properties</h2>
        <dl className={css.resourceInfo}>
          {customPropertyMap.getEntryList().map(k => (
            <div className={css.field} key={k[0]} data-testid='resource-info-property'>
              <dt className={css.term} data-testid='resource-info-property-key'>
                {k[0]}
              </dt>
              <dd className={css.value} data-testid='resource-info-property-value'>
                {customPropertyMap &&
                  prettyPrintValue(getMetadataValue(customPropertyMap.get(k[0])))}
              </dd>
            </div>
          ))}
        </dl>
      </section>
    );
  }
}

function prettyPrintValue(
  value: string | number | Struct | undefined,
): JSX.Element | number | string {
  if (value == null) {
    return '';
  }
  if (typeof value === 'string') {
    return prettyPrintJsonValue(value);
  }
  if (typeof value === 'number') {
    return value;
  }
  // value is Struct
  const jsObject = value.toJavaScript();
  // When Struct is converted to js object, it may contain a top level "struct"
  // or "list" key depending on its type, but the key is meaningless and we can
  // omit it in visualization.
  return <pre>{JSON.stringify(jsObject?.struct || jsObject?.list || jsObject, null, 2)}</pre>;
}

function prettyPrintJsonValue(value: string): JSX.Element | string {
  try {
    const jsonValue = JSON.parse(value);
    return <pre>{JSON.stringify(jsonValue, null, 2)}</pre>;
  } catch {
    // not JSON, return directly
    return value;
  }
}
