import * as jsYaml from 'js-yaml';
import { PackageTemplate } from '../api/pipeline_package';
import {
  Arguments as ArgoTemplateStepArguments,
  Parameter as ArgoTemplateStepParameter,
  Workflow as ArgoTemplate,
  WorkflowStep as ArgoTemplateStep,
} from '../model/argo_template';

function replacePlaceholders(path: string, baseOutputPath: string, jobId: string): string {
  return path
    .replace(/{{inputs.parameters.output}}/, baseOutputPath)
    .replace(/{{workflow.name}}/, jobId);
}

export interface OutputInfo {
  index?: number;
  path: string;
  step: string;
}

export function parseTemplateOuputPaths(
    packageTemplate: PackageTemplate,
    baseOutputPath: string,
    jobId: string
  ): OutputInfo[] {
  if (!packageTemplate.template) {
    throw new Error('Failed to load the package template');
  }
  const argoTemplate = jsYaml.safeLoad(packageTemplate.template) as ArgoTemplate;
  // TODO: Support templates with no entrypoint (only one template element)
  if (!argoTemplate) {
    throw new Error('Failed to load the workflow argo template');
  }
  if (!argoTemplate.spec) {
    throw new Error('Workflow argo template does not contain a spec');
  }
  const spec = argoTemplate.spec;
  if (!spec.entrypoint) {
    throw new Error('Spec does not contain an entrypoint');
  }
  const entryPoint = spec.entrypoint;

  if (!spec.templates) {
    throw new Error('Spec does not contain any templates');
  }
  const entryTemplate = spec.templates.filter((t) => t.name === entryPoint)[0];

  if (!entryTemplate) {
    throw new Error('Could not find template for entrypoint: ' + entryPoint);
  }

  // Steps can be nested twice (because of Argo's double dash convention) or just once
  // so, flatten it first.
  const steps = [].concat.apply([], entryTemplate.steps) as ArgoTemplateStep[];

  if (!steps) {
    return [];
  }

  return steps.map((step) => {
    if (Array.isArray(step)) {
      step = step[0];
    }
    const args = (step.arguments as ArgoTemplateStepArguments);
    const params = args.parameters as ArgoTemplateStepParameter[];
    const outputParam = params.filter((p) => p.name === 'output');
    const path = outputParam && outputParam.length === 1 ?
        replacePlaceholders(outputParam[0].value as string, baseOutputPath, jobId) : '';
    return {
      path,
      step: step.name as string,
    };
  }).filter((p) => !!p.path);
}
