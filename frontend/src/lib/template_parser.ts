import * as jsYaml from 'js-yaml';
import {
  Arguments as ArgoTemplateStepArguments,
  Parameter as ArgoTemplateStepParameter,
  Workflow as ArgoTemplate,
  WorkflowStep as ArgoTemplateStep,
} from '../model/argo_template';

function replacePlaceholders(path: string, baseOutputPath: string, jobId: string) {
  return path
    .replace(/{{inputs.parameters.output}}/, baseOutputPath)
    .replace(/{{workflow.name}}/, jobId);
}

export function parseTemplateOuputPaths(templateYaml: string, baseOutputPath: string, jobId: string) {
  const argoTemplate = jsYaml.safeLoad(templateYaml) as ArgoTemplate;

  // TODO: Write a proper parser with validation for the YAML
  const entryPoint = argoTemplate.spec.entrypoint;
  const entryTemplate = argoTemplate.spec.templates.filter((t) => t.name === entryPoint)[0];
  // Steps can be nested twice (because of Argo's double dash convention) or just once
  const steps = entryTemplate.steps as Array<ArgoTemplateStep | ArgoTemplateStep[]>;

  const outputPaths: Array<{ step: string, path: string }> = steps.map((step) => {
    if (Array.isArray(step)) {
      step = step[0];
    }
    const args = (step.arguments as ArgoTemplateStepArguments);
    const params = args.parameters as ArgoTemplateStepParameter[];
    const outputParam = params.filter((p) => p.name === 'output');
    return {
      path: outputParam && outputParam.length === 1 ? outputParam[0].value as string : '',
      step: step.name as string,
    };
  }).filter((p) => !!p.path);

  return outputPaths.map((p) => replacePlaceholders(p.path, baseOutputPath, jobId));
}
