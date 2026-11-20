"""Exercise the KFP Workflow drain check with representative Argo responses."""

import unittest

from check_workflow_drain import active_workflows


class WorkflowDrainTest(unittest.TestCase):
    def test_only_active_workflows_owned_by_the_installation_block_the_drain(
        self,
    ) -> None:
        runs = [{'run_id': 'local-run', 'namespace': 'kubeflow'}]

        def workflow(
            name: str,
            run_id: str | None,
            namespace: str = 'kubeflow',
            phase: str | None = 'Running',
        ) -> dict[str, dict[str, object]]:
            return {
                'metadata': {
                    'name': name,
                    'namespace': namespace,
                    'labels': {'pipeline/runid': run_id} if run_id else {},
                },
                'status': {'phase': phase},
            }

        workflows = {
            'items': [
                workflow('ordinary-argo', None),
                workflow('other-installation', 'other-run'),
                workflow('other-namespace', 'local-run', namespace='other'),
                workflow('finished', 'local-run', phase='Succeeded'),
                workflow('still-running', 'local-run'),
                workflow('not-started', 'local-run', phase=None),
            ]
        }
        self.assertEqual(
            [('kubeflow', 'not-started'), ('kubeflow', 'still-running')],
            active_workflows(runs, workflows),
        )

    def test_missing_namespace_for_known_run_refuses_check(self) -> None:
        runs = [{'run_id': 'local-run', 'namespace': ''}]
        workflows = {
            'items': [{'metadata': {'labels': {'pipeline/runid': 'local-run'}}}]
        }
        with self.assertRaisesRegex(ValueError, 'has no namespace'):
            active_workflows(runs, workflows)


if __name__ == '__main__':
    unittest.main()
