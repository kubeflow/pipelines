// tslint:disable:object-literal-sort-keys
export default {
  metadata: {
    name: 'coinflip-error-nklng2',
    namespace: 'default',
    // tslint:disable-next-line:max-line-length
    selfLink: '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/coinflip-heads-c085010d-771a-4cdf-979c-257e991501b5',
    uid: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
    resourceVersion: '10527150',
    creationTimestamp: '2018-06-11T22:49:26Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Failed'
    }
  },
  spec: {
    templates: [
      {
        name: 'coinflip',
        inputs: {},
        outputs: {},
        metadata: {},
        steps: [
          [
            {
              name: 'heads',
              template: 'heads',
              arguments: {},
              when: '{{steps.flip-coin.outputs.result}} == heads'
            }
          ]
        ]
      },
      {
        name: 'heads',
        inputs: {},
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'alpine:3.6',
          command: [
            'sh',
            '-c'
          ],
          args: [
            'echo "it was heads"'
          ],
          resources: {}
        }
      }
    ],
    entrypoint: 'coinflip',
    arguments: {}
  },
  status: {
    phase: 'Failed',
    startedAt: '2018-06-11T22:49:26Z',
    finishedAt: '2018-06-11T22:49:26Z',
    // tslint:disable-next-line:max-line-length
    message: 'invalid spec: templates.coinflip.steps[0].heads failed to resolve {{steps.flip-coin.outputs.result}}'
  }
};
