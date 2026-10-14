import io,json,sys,time,unittest
from pathlib import Path
from unittest.mock import patch
from capture import decode_line,observed_users,project_file,snapshot
from join import join
from project import project
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from provision import audit_configuration
class EvidenceTests(unittest.TestCase):
 def test_kind_metadata_audit_excludes_pressure_and_has_readonly_policy_mount(self):
  policy,config=audit_configuration(Path('/tmp/owned'),'kfp-apf-representative-test')
  self.assertEqual(policy['rules'][0]['level'],'Metadata')
  self.assertEqual(len(policy['rules'][0]['userGroups']),5)
  self.assertFalse(any('apf-background' in g or 'apf-competitor' in g for g in policy['rules'][0]['userGroups']))
  node=config['nodes'][0];self.assertTrue(node['extraMounts'][0]['readOnly'])
  args={v['name']:v['value'] for v in json.loads(node['kubeadmConfigPatches'][0])['apiServer']['extraArgs']}
  self.assertEqual(args['logging-format'],'json');self.assertEqual(args['v'],'3');self.assertEqual(args['audit-log-maxsize'],'16')
 def test_actual_deployment_and_pod_accounts_only(self):
  deployments={'items':[{'metadata':{'namespace':'kubeflow'},'spec':{'template':{'spec':{'serviceAccountName':'actual-api'}}}}]}
  pods={'items':[{'metadata':{'namespace':'tenant'},'spec':{'serviceAccountName':'driver'}},{'metadata':{'namespace':'apf-background'},'spec':{'serviceAccountName':'pressure'}}]}
  self.assertEqual(observed_users(deployments,pods,{'kubeflow','tenant'}),{'system:serviceaccount:kubeflow:actual-api','system:serviceaccount:tenant:driver'})
 def test_identity_snapshot_uses_projected_columns_not_full_specs(self):
  calls=[]
  class Owner:
   def k(self,*args,**kwargs):
    calls.append(args)
    if args[1]=='deployments':return b'kubeflow\tml-pipeline\tactual-api\n'
    if args[1]=='pods':return b'tenant\tdriver-pod\tactual-driver\n'
    return b'{"items":[]}'
  result=snapshot(Owner(),{'kubeflow','tenant'})
  self.assertEqual(result['api_user'],'system:serviceaccount:kubeflow:actual-api')
  self.assertIn('system:serviceaccount:tenant:actual-driver',result['users'])
  self.assertTrue(calls[0][-1].startswith('jsonpath='));self.assertTrue(calls[1][-1].startswith('jsonpath='))
 def test_stream_projection_discards_secret_raw_fields(self):
  raw={'msg':'HTTP','audit-ID':'12345678','apf_fs':'f','apf_pl':'p','resp':200,'requestURI':'SECRET','Authorization':'BEARER'}
  line=b'2026-01-01T00:00:00Z stdout F '+json.dumps(raw).encode()+b'\n'
  self.assertEqual(decode_line(line,'http'),raw)
  script='import sys;sys.stdout.buffer.write('+repr(line)+')'
  output=io.StringIO();count,bad=project_file([sys.executable,'-c',script],'http',set(),output,[0,time.monotonic()+10])
  self.assertEqual((count,bad),(1,0));self.assertNotIn('SECRET',output.getvalue());self.assertNotIn('BEARER',output.getvalue())
  with patch('capture.LIMIT',10):
   with self.assertRaisesRegex(RuntimeError,'bound'):project_file([sys.executable,'-c',script],'http',set(),io.StringIO(),[0,time.monotonic()+10])
 def test_actual_audit_user_join_rejects_ambiguous_prefix(self):
  audit=[{'audit_id':'12345678-one','user':'system:serviceaccount:kubeflow:actual-api','verb':'create','group':'authentication.k8s.io','resource':'tokenreviews'}]
  http=[{'audit_prefix':'12345678','flow_name':'f','priority_name':'p'}]
  fs={'f':{'metadata':{'uid':'fuid'},'spec':{'priorityLevelConfiguration':{'name':'p'}}}};pl={'p':{'metadata':{'uid':'puid'},'spec':{'type':'Limited'}}}
  self.assertEqual(join(audit,http,fs,pl)['records'][0]['priority_uid'],'puid')
  audit.append(dict(audit[0],audit_id='12345678-two'))
  self.assertEqual(join(audit,http,fs,pl)['ambiguous'],2)
if __name__=='__main__':unittest.main()
