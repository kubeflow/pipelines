import { getRecurringRunsUnavailableReason } from './BuildInfo';
import { V2GetHealthzResponseFromJSON } from 'src/apisv2/healthz';

describe('recurring-run deployment capabilities', () => {
  it('preserves explicit false through the generated health response decoder', () => {
    const response = V2GetHealthzResponseFromJSON({
      capabilities: {
        recurring_runs: { supported: false, reason: 'Controller unavailable' },
      },
    });
    expect(getRecurringRunsUnavailableReason(response)).toBe('Controller unavailable');
    expect(getRecurringRunsUnavailableReason(V2GetHealthzResponseFromJSON({}))).toBeUndefined();
  });

  it.each([undefined, {}, { capabilities: {} }, { capabilities: { recurring_runs: {} } }])(
    'retains legacy behavior when support is not advertised: %j',
    (buildInfo) => {
      expect(getRecurringRunsUnavailableReason(buildInfo)).toBeUndefined();
    },
  );

  it('does not infer support from deployment or pipeline storage', () => {
    expect(getRecurringRunsUnavailableReason({ pipelineStore: 'database' })).toBeUndefined();
    expect(
      getRecurringRunsUnavailableReason({
        pipelineStore: 'kubernetes',
        capabilities: { recurring_runs: { supported: false, reason: 'Controller unavailable' } },
      }),
    ).toBe('Controller unavailable');
  });

  it('enables supported deployments and provides a fallback explanation for explicit false', () => {
    expect(
      getRecurringRunsUnavailableReason({
        capabilities: { recurring_runs: { supported: true, reason: 'Ignored' } },
      }),
    ).toBeUndefined();
    expect(
      getRecurringRunsUnavailableReason({ capabilities: { recurring_runs: { supported: false } } }),
    ).toContain('not supported');
  });
});
