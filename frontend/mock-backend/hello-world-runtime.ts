// tslint:disable:object-literal-sort-keys
export default {
  metadata: {
    name: 'hello-world-7sm94',
    generateName: 'hello-world-',
    namespace: 'default',
    selfLink: '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-7sm94',
    uid: '3ac97065-691d-11e8-802f-42010af0033a',
    resourceVersion: '1322',
    creationTimestamp: '2018-06-06T00:04:49Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded'
    }
  },
  spec: {
    templates: [
      {
        name: 'whalesay1',
        inputs: {},
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'docker/whalesay:latest',
          command: [
            'cowsay'
          ],
          args: [
            '{{workflow.parameters.message}}'
          ],
          resources: {}
        }
      }
    ],
    entrypoint: 'whalesay1',
    arguments: {
      parameters: [
        {
          name: 'message',
          value: 'hello world'
        }
      ]
    }
  },
  status: {
    phase: 'Succeeded',
    startedAt: '2018-06-06T00:04:49Z',
    finishedAt: '2018-06-06T00:05:23Z',
    nodes: {
      'hello-world-7sm94': {
        id: 'hello-world-7sm94',
        name: 'hello-world-7sm94',
        displayName: 'hello-world-7sm94',
        type: 'Pod',
        templateName: 'whalesay1',
        phase: 'Succeeded',
        startedAt: '2018-06-06T00:04:49Z',
        finishedAt: '2018-06-06T00:05:23Z'
      }
    }
  }
};
