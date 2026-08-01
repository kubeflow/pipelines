// Demonstrates the pod lifecycle failure color-coding and cause/fix tooltips
// added to Graph/StatusUtils. Run `npm run storybook` and open
// Demo/PodLifecycleFailureGraph to see it interactively.
import { Meta, StoryObj } from '@storybook/react';
import RunGraph from '../components/Graph';
import WorkflowParser from '../lib/WorkflowParser';

const workflow = {
  metadata: { name: 'demo-pipeline' },
  status: {
    nodes: {
      root: {
        displayName: 'demo-pipeline',
        id: 'root',
        name: 'demo-pipeline',
        phase: 'Failed',
        type: 'Steps',
        children: ['load-data', 'train-model', 'evaluate', 'deploy'],
      },
      'load-data': {
        displayName: 'load-data',
        id: 'load-data',
        name: 'load-data',
        phase: 'Succeeded',
        type: 'Pod',
      },
      'train-model': {
        displayName: 'train-model',
        id: 'train-model',
        name: 'train-model',
        phase: 'Failed',
        type: 'Pod',
        message: 'container terminated with reason OOMKilled, exit code 137',
      },
      evaluate: {
        displayName: 'evaluate',
        id: 'evaluate',
        name: 'evaluate',
        phase: 'Failed',
        type: 'Pod',
        message: '0/3 nodes are available: 3 Insufficient cpu. Unschedulable',
      },
      deploy: {
        displayName: 'deploy',
        id: 'deploy',
        name: 'deploy',
        phase: 'Failed',
        type: 'Pod',
        message: 'pod was Preempted to make room for a higher priority pod',
      },
    },
  },
} as any;

const graph = WorkflowParser.createRuntimeGraph(workflow, undefined);

const meta: Meta<typeof RunGraph> = {
  title: 'Demo/PodLifecycleFailureGraph',
  component: RunGraph,
};
export default meta;

type Story = StoryObj<typeof RunGraph>;

export const PodLifecycleFailures: Story = {
  args: { graph },
};
