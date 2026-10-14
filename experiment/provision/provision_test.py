import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import provision

class GuardTests(unittest.TestCase):
    def test_cluster_allowlist_rejects_standalone_and_production(self):
        for name in ('production', 'kfp-apf-experiment', 'kfp'):
            with self.assertRaises(RuntimeError):
                provision.validate({'source_sha': provision.SOURCE, 'cluster': name})

    def test_node_tag_is_not_immutable_identity(self):
        with self.assertRaises(RuntimeError):
            provision.validate({'source_sha': provision.SOURCE, 'cluster': 'kfp-apf-representative-test', 'node_image': 'kindest/node:v1.36.1'})

    def test_namespace_uid_prevents_adoption(self):
        with tempfile.TemporaryDirectory() as d:
            p = provision.Provision({'cluster': 'kfp-apf-representative-test'}, d)
            provision.save(p.marker, {'cluster': p.c['cluster'], 'source_sha': provision.SOURCE, 'namespace_uid': 'owned'})
            with patch.object(p, 'k', return_value=b'replacement'):
                with self.assertRaises(RuntimeError):
                    p.guard()

    def test_cleanup_requires_capture_and_never_calls_kind_on_refusal(self):
        with tempfile.TemporaryDirectory() as d:
            p = provision.Provision({'cluster': 'kfp-apf-representative-test'}, d)
            with patch.object(p, 'guard'), patch('provision.call') as runner:
                with self.assertRaises(RuntimeError):
                    p.cleanup()
                runner.assert_not_called()

    def test_child_errors_do_not_expose_output(self):
        result = subprocess.CompletedProcess(['fake'], 1, stdout=b'secret-token', stderr=b'secret-token')
        with patch('subprocess.run', return_value=result):
            with self.assertRaises(RuntimeError) as e:
                provision.call(['fake'])
        self.assertNotIn('secret-token', str(e.exception))


class ProvenanceTests(unittest.TestCase):
    def make_image(self, directory, revision=provision.SOURCE):
        import hashlib
        import io
        import tarfile
        import zipfile
        d = Path(directory)
        config = json.dumps({'architecture': 'amd64', 'config': {}}).encode()
        digest = hashlib.sha256(config).hexdigest()
        tar = d / 'image.tar'
        with tarfile.open(tar, 'w') as archive:
            for name, data in [('manifest.json', json.dumps([{'Config': digest + '.json'}]).encode()), (digest + '.json', config)]:
                info = tarfile.TarInfo(name)
                info.size = len(data)
                archive.addfile(info, io.BytesIO(data))
        artifact_zip = d / 'artifact.zip'
        with zipfile.ZipFile(artifact_zip, 'w') as archive:
            archive.write(tar, 'image.tar')
        artifact = d / 'artifact.json'
        provision.save(artifact, {'id': 456, 'name': 'apiserver', 'digest': 'sha256:' + provision.sha(artifact_zip), 'expired': False, 'workflow_run': {'id': 123, 'head_sha': provision.HEAD}})
        run = d / 'run.json'
        provision.save(run, {'id': 123, 'head_sha': provision.HEAD, 'conclusion': 'success', 'path': '.github/workflows/api-server-tests.yml', 'repository': {'full_name': 'kubeflow/pipelines'}})
        job = d / 'job.json'
        provision.save(job, {'run_id': 123, 'conclusion': 'success', 'name': 'build / image-build (apiserver, backend/Dockerfile, .)', 'steps': [{'name': 'Upload artifact', 'conclusion': 'success'}, {'name': 'Build and save Docker image', 'conclusion': 'success'}]})
        log = d / 'job.log'
        log.write_text(f'image-builds.yml@refs/pull/14818/merge ({provision.SOURCE})\n2026Z {provision.SOURCE}\nactions/runs/123/artifacts/456')
        commit = d / 'commit.json'
        provision.save(commit, {'sha': provision.SOURCE, 'parents': [{'sha': provision.HEAD}]})
        return {'component': 'apiserver', 'producer_job_metadata': str(job), 'producer_job_log': str(log), 'source_commit_metadata': str(commit), 'image_id': 'sha256:' + digest, 'archive': str(tar), 'archive_sha256': provision.sha(tar), 'artifact_zip': str(artifact_zip), 'archive_member': 'image.tar', 'artifact_metadata': str(artifact), 'producer_run_metadata': str(run)}

    def test_bound_archive_passes_and_existing_image_id_cannot_mask_wrong_tar(self):
        with tempfile.TemporaryDirectory() as d:
            image = self.make_image(d)
            provision.verify_image(image)
            image['image_id'] = 'sha256:' + '0' * 64
            with self.assertRaisesRegex(RuntimeError, 'tar config'):
                provision.verify_image(image)

    def test_unsigned_source_claim_and_zip_tampering_refused(self):
        with tempfile.TemporaryDirectory() as d:
            image = self.make_image(d, revision='wrong')
            provision.verify_image(image)  # Existing CI images legitimately lack OCI revision labels.
            Path(image['artifact_zip']).write_bytes(b'tampered')
            with self.assertRaisesRegex(RuntimeError, 'ZIP digest'):
                provision.verify_image(image)

    def test_old_producer_head_refused(self):
        with tempfile.TemporaryDirectory() as d:
            image = self.make_image(d)
            run = json.loads(Path(image['producer_run_metadata']).read_text())
            run['head_sha'] = 'old'
            provision.save(image['producer_run_metadata'], run)
            with self.assertRaisesRegex(RuntimeError, 'producer metadata'):
                provision.verify_image(image)


class TransportReadinessTests(unittest.TestCase):
    def test_all_redirect_codes_refuse_bearer_forwarding(self):
        import urllib.request
        import urllib.error
        request = urllib.request.Request('http://127.0.0.1:18888/apis/v2beta1/experiments', headers={'Authorization': 'Bearer secret'}, data=b'{}')
        for code in (301, 302, 303, 307, 308):
            with self.assertRaises(urllib.error.URLError):
                provision.NoRedirect().redirect_request(request, None, code, 'redirect', {}, 'https://other.invalid/')

    def test_post_installs_no_redirect_handler(self):
        with patch('provision.urllib.request.build_opener') as opener:
            opener.return_value.open.side_effect = provision.urllib.error.URLError('secret')
            with self.assertRaises(RuntimeError) as error:
                provision.Provision.post('http://127.0.0.1:18888/', 'secret', b'{}', 'application/json')
            self.assertIsInstance(opener.call_args.args[0], provision.NoRedirect)
            self.assertNotIn('secret', str(error.exception))

    def test_iam_requires_three_consecutive_successes_and_never_reads_logs(self):
        p = provision.Provision({'cluster': 'kfp-apf-representative-test'}, '/private/tmp/test-unused')
        with patch.object(p, 'k', side_effect=[b'', RuntimeError(), b'', b'', b'']) as k, patch('provision.time.sleep'):
            p.wait_iam()
        self.assertEqual(k.call_count, 5)
        for invocation in k.call_args_list:
            self.assertIn('exec', invocation.args)
            self.assertNotIn('logs', invocation.args)

    def test_rendered_hook_frontend_and_all_controller_readiness(self):
        import sys
        from types import SimpleNamespace
        objects = [{'kind': kind, 'metadata': {'name': name, 'namespace': 'kubeflow'}, 'spec': {'template': {'spec': {'containers': [{'name': 'profile-controller', 'env': [{'name': 'KEEP', 'value': 'yes'}]}]}}}} for kind, name in [('Deployment', 'kubeflow-pipelines-profile-controller'), ('Deployment', 'mysql'), ('Deployment', 'workflow-controller'), ('StatefulSet', 'metacontroller'), ('StatefulSet', 'seaweedfs')]]
        with tempfile.TemporaryDirectory() as d:
            p = provision.Provision({'cluster': 'kfp-apf-representative-test', 'images': {}, 'qps': 20}, d)
            with patch.dict(sys.modules, {'yaml': SimpleNamespace(safe_load_all=lambda _: objects)}), patch('provision.validate'), patch.object(p, 'guard'), patch.object(p, 'apply'), patch.object(p, 'wait_iam') as iam, patch.object(p, 'k', return_value=b'{}') as k:
                p.deploy()
            env = {e['name']: e['value'] for e in objects[0]['spec']['template']['spec']['containers'][0]['env']}
            self.assertEqual(env, {'KEEP': 'yes', 'FRONTEND_IMAGE': 'kfp-apf-local/frontend', 'FRONTEND_TAG': provision.SOURCE})
            for obj in objects:
                target = obj['kind'].lower() + '/' + obj['metadata']['name']
                self.assertTrue(any(target in c.args and 'rollout' in c.args for c in k.call_args_list))
            iam.assert_called_once()

if __name__ == '__main__':
    unittest.main()
