/*
 * Copyright 2018 The Kubeflow Authors
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

import { csvParseRows } from 'd3-dsv';
import { ConfusionMatrixConfig } from '../components/viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../components/viewers/MarkdownViewer';
import { PagedTableConfig } from '../components/viewers/PagedTable';
import { ROCCurveConfig } from '../components/viewers/ROCCurve';
import { TensorboardViewerConfig } from '../components/viewers/Tensorboard';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { Apis } from '../lib/Apis';
import { errorToMessage, logger } from './Utils';
import WorkflowParser, { StoragePath } from './WorkflowParser';
import { parseArtifactFileLocation } from './v2/ArtifactFileUtils';
export interface PlotMetadata {
  format?: 'csv';
  header?: string[];
  labels?: string[];
  predicted_col?: string;
  schema?: Array<{ type: string; name: string }>;
  source: string;
  storage?: 'gcs' | 'inline';
  target_col?: string;
  pod_template_spec?: any; // only available for tensorboard
  image?: string; // only available for tensorboard
  type: PlotType;
}

type PlotMetadataContent = Omit<PlotMetadata, 'type'>;

export interface OutputMetadata {
  outputs: PlotMetadata[];
}

export interface OutputArtifactLoadOptions {
  artifactUriQuery?: string;
  providerInfo?: string;
  throwOnError?: boolean;
}

export interface OutputArtifactLoadResult {
  configs: ViewerConfig[];
  errors: string[];
}

type SourceContentGetter = (source: string, storage?: PlotMetadata['storage']) => Promise<string>;

export class OutputArtifactLoader {
  public static async load(
    outputPath: StoragePath,
    namespace?: string,
    options: OutputArtifactLoadOptions = {},
  ): Promise<ViewerConfig[]> {
    const result = await this.loadResult(outputPath, namespace, options);
    if (options.throwOnError && result.errors.length) {
      throw new Error(result.errors.join('\n'));
    }
    return result.configs;
  }

  public static async loadResult(
    outputPath: StoragePath,
    namespace?: string,
    options: OutputArtifactLoadOptions = {},
  ): Promise<OutputArtifactLoadResult> {
    let plotMetadataList: PlotMetadata[] = [];
    try {
      const metadataFile = await Apis.readFile({
        path: outputPath,
        namespace,
        artifactUriQuery: options.artifactUriQuery,
        providerInfo: options.providerInfo,
      });
      if (metadataFile) {
        try {
          plotMetadataList = OutputArtifactLoader.parseOutputMetadataInJson(
            metadataFile,
            outputPath.key,
          );
        } catch (e) {
          // This is a hack which only works on scenario for html/tensorboard, but not markdown.
          // Because podTemplateSpec is escaped twice before writing to file. There are '\' before
          // each `"` in podTemplateSpec.
          // https://github.com/kubeflow/pipelines/issues/5830
          const editMetadataFile = metadataFile.replace(/(\r\n|\n|\r|\\)/gm, '');
          plotMetadataList = OutputArtifactLoader.parseOutputMetadataInJson(
            editMetadataFile,
            outputPath.key,
          );
        }
      }
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      logger.error('Error loading run outputs:', errorMessage);
      if (options.throwOnError) {
        throw err;
      }
    }

    const getSourceContent: SourceContentGetter = async (source, storage) =>
      await readSourceContent(source, storage, namespace);

    const results = await Promise.allSettled(
      plotMetadataList.map(async (metadata) => {
        switch (metadata.type) {
          case PlotType.CONFUSION_MATRIX:
            return await this.buildConfusionMatrixConfig(metadata, getSourceContent);
          case PlotType.MARKDOWN:
            return await this.buildMarkdownViewerConfig(metadata, getSourceContent);
          case PlotType.TABLE:
            return await this.buildPagedTableConfig(metadata, getSourceContent);
          case PlotType.TENSORBOARD:
            return await this.buildTensorboardConfig(metadata, namespace);
          case PlotType.WEB_APP:
            return await this.buildHtmlViewerConfig(metadata, getSourceContent);
          case PlotType.ROC:
            return await this.buildRocCurveConfig(metadata, getSourceContent);
          default:
            logger.error('Unknown plot type: ' + metadata.type);
            return null;
        }
      }),
    );

    const configs: ViewerConfig[] = [];
    const errors: string[] = [];
    for (const result of results) {
      if (result.status === 'fulfilled') {
        if (result.value) {
          configs.push(result.value);
        }
      } else {
        const message = await errorToMessage(result.reason);
        logger.error('Error loading run output:', message);
        errors.push(message);
      }
    }
    return { configs, errors };
  }

  private static parseOutputMetadataInJson(fileContent: string, key: string): PlotMetadata[] {
    try {
      const plotMetadataList = (JSON.parse(fileContent) as OutputMetadata).outputs;
      if (plotMetadataList === undefined) {
        throw new Error('"outputs" field required by not found on metadata file');
      }
      return plotMetadataList;
    } catch (e) {
      logger.error(`Could not parse metadata file at: ${key}. Error: ${e}`);
      throw new Error(`Could not parse metadata file at: ${key}. Error: ${e}`, { cause: e });
    }
  }

  public static async buildConfusionMatrixConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<ConfusionMatrixConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.labels) {
      throw new Error('Malformed metadata, property "labels" is required.');
    }
    if (!metadata.schema) {
      throw new Error('Malformed metadata, property "schema" missing.');
    }
    if (!Array.isArray(metadata.schema)) {
      throw new Error('"schema" must be an array of {"name": string, "type": string} objects');
    }

    const content = await getSourceContent(metadata.source, metadata.storage);
    const csvRows = csvParseRows(content.trim());
    const labels = metadata.labels;
    const labelIndex: { [label: string]: number } = {};
    let index = 0;
    labels.forEach((l) => {
      labelIndex[l] = index++;
    });

    if (labels.length ** 2 !== csvRows.length) {
      throw new Error(
        `Data dimensions ${csvRows.length} do not match the number of labels passed ${labels.length}`,
      );
    }

    const data = Array.from(Array(labels.length), () => new Array(labels.length));
    csvRows.forEach(([labelX, labelY, count]) => {
      const i = labelIndex[labelX.trim()];
      const j = labelIndex[labelY.trim()];
      // Note: data[i][j] means data(i, j) i on x-axis, j on y-axis
      data[i][j] = Number.parseInt(count, 10);
    });

    const columnNames = metadata.schema.map((r) => {
      if (!r.name) {
        throw new Error('Each item in the "schema" array must contain a "name" field');
      }
      return r.name;
    });
    const axes = [columnNames[0], columnNames[1]];

    return {
      axes,
      data,
      labels,
      type: PlotType.CONFUSION_MATRIX,
    };
  }

  public static async buildPagedTableConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<PagedTableConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.header) {
      throw new Error('Malformed metadata, property "header" is required.');
    }
    if (!metadata.format) {
      throw new Error('Malformed metadata, property "format" is required.');
    }
    let data: string[][];
    const labels = metadata.header || [];
    const content = await getSourceContent(metadata.source, metadata.storage);

    switch (metadata.format) {
      case 'csv':
        data = csvParseRows(content.trim()).map((r) => r.map((c) => c.trim()));
        break;
      default:
        throw new Error('Unsupported table format: ' + metadata.format);
    }

    return {
      data,
      labels,
      type: PlotType.TABLE,
    };
  }

  public static async buildTensorboardConfig(
    metadata: PlotMetadataContent,
    namespace?: string,
  ): Promise<TensorboardViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!namespace) {
      throw new Error('Namespace is required.');
    }
    WorkflowParser.parseStoragePath(metadata.source);
    return {
      type: PlotType.TENSORBOARD,
      url: metadata.source,
      namespace,
      podTemplateSpec: metadata.pod_template_spec,
      image: metadata.image,
    };
  }

  public static async buildHtmlViewerConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<HTMLViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    return {
      htmlContent: await getSourceContent(metadata.source, metadata.storage),
      type: PlotType.WEB_APP,
    };
  }

  public static async buildMarkdownViewerConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<MarkdownViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    return {
      markdownContent: await getSourceContent(metadata.source, metadata.storage),
      type: PlotType.MARKDOWN,
    };
  }

  public static async buildRocCurveConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<ROCCurveConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.schema) {
      throw new Error('Malformed metadata, property "schema" is required.');
    }
    if (!Array.isArray(metadata.schema)) {
      throw new Error('Malformed schema, must be an array of {"name": string, "type": string}');
    }

    const content = await getSourceContent(metadata.source, metadata.storage);
    const stringData = csvParseRows(content.trim());

    const fprIndex = metadata.schema.findIndex((field) => field.name === 'fpr');
    if (fprIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "fpr"');
    }
    const tprIndex = metadata.schema.findIndex((field) => field.name === 'tpr');
    if (tprIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "tpr"');
    }
    const thresholdIndex = metadata.schema.findIndex((field) => field.name.startsWith('threshold'));
    if (thresholdIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "threshold"');
    }

    const dataset = stringData.map((row) => ({
      label: row[thresholdIndex].trim(),
      x: +row[fprIndex],
      y: +row[tprIndex],
    }));

    return {
      data: dataset,
      type: PlotType.ROC,
    };
  }
}

async function readSourceContent(
  source: PlotMetadata['source'],
  storage: PlotMetadata['storage'] | undefined,
  namespace: string | undefined,
): Promise<string> {
  if (storage === 'inline') {
    return source;
  }
  const location = parseArtifactFileLocation(source);
  return await Apis.readFile({
    path: location.path,
    namespace,
    artifactUriQuery: location.artifactUriQuery,
  });
}

export const TEST_ONLY = {
  readSourceContent,
};
