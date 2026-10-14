import unittest,json,os,sys,tempfile
from pathlib import Path
from unittest.mock import patch
from coordinator import Owned
from prepare_background import resources,classification,bulk_resources,verify_seed,verify_list,read_seed
class BackgroundTests(unittest.TestCase):
 def test_role_is_namespace_scoped_named_configmap_only(self):
  objects=resources();role=next(o for o in objects if o['kind']=='Role')
  self.assertEqual(role['rules'],[{'apiGroups':[''],'resources':['configmaps'],'resourceNames':['probe'],'verbs':['get','update']},{'apiGroups':[''],'resources':['configmaps'],'verbs':['list']}])
  self.assertFalse(any(o['kind'].startswith('Cluster') for o in objects))
  binding=next(o for o in objects if o['kind']=='RoleBinding')
  self.assertEqual(binding['subjects'][0]['name'],'pressure');self.assertEqual(binding['subjects'][0]['namespace'],'apf-background')
 def test_actual_headers_must_resolve_same_limited_priority(self):
  fs={'items':[{'metadata':{'uid':'f'},'spec':{'priorityLevelConfiguration':{'name':'shared'}}}]}
  pl={'items':[{'metadata':{'uid':'p','name':'shared'},'spec':{'type':'Limited'}}]}
  result=classification('f','p',fs,pl,{'priority_uid':'p'})
  self.assertTrue(result['quiet_shared_priority_verified']);self.assertFalse(result['actual_kfp_classification_verified'])
  for flow,priority in [(None,'p'),('wrong','p'),('f','wrong')]:
   with self.assertRaises(RuntimeError):classification(flow,priority,fs,pl,{'priority_uid':'p'})
  with self.assertRaises(RuntimeError):classification('f','p',fs,pl,{'priority_uid':'other'})
  pl['items'][0]['spec']['type']='Exempt'
  with self.assertRaises(RuntimeError):classification('f','p',fs,pl,{'priority_uid':'p'})
 def test_bulk_seed_exact_count_bytes_and_identity(self):
  objects=bulk_resources();proof=verify_seed(objects)
  self.assertEqual(proof['bulk_object_count'],1024);self.assertEqual(proof['total_payload_bytes'],1048576)
  self.assertEqual(objects[0]['metadata']['name'],'bulk-0000');self.assertEqual(objects[-1]['metadata']['name'],'bulk-1023')
  with self.assertRaisesRegex(RuntimeError,'count or names'):verify_seed(objects[:-1])
  objects[0]['data']['fixed']='x'*1025
  with self.assertRaisesRegex(RuntimeError,'payload'):verify_seed(objects)
 def test_authenticated_list_requires_all_seed_objects_without_pagination(self):
  listing={'items':bulk_resources()+[{'metadata':{'name':'probe'}}]}
  self.assertEqual(verify_list(listing),1025)
  listing['metadata']={'continue':'opaque'}
  with self.assertRaisesRegex(RuntimeError,'incomplete'):verify_list(listing)
  listing={'items':[{'metadata':{'name':'probe'}}]*2049}
  with self.assertRaisesRegex(RuntimeError,'item bound'):verify_list(listing)
 def test_executable_seed_read_exceeds_default_but_stays_bounded(self):
  with tempfile.TemporaryDirectory() as tmp:
   folder=Path(tmp);payload=folder/'output.json'
   payload.write_text(json.dumps({'items':bulk_resources()}))
   self.assertGreater(payload.stat().st_size,1<<20)
   kubectl=folder/'kubectl'
   kubectl.write_text('#!'+sys.executable+'\nimport pathlib,sys\nsys.stdout.buffer.write(pathlib.Path('+repr(str(payload))+').read_bytes())\n')
   kubectl.chmod(0o700)
   owner=Owned({'state':tmp,'cluster':'kfp-apf-representative-test'})
   with patch.dict(os.environ,{'PATH':tmp+os.pathsep+os.environ.get('PATH','')}):
    with self.assertRaisesRegex(RuntimeError,'output limit'):owner.k('get','configmaps')
    proof=read_seed(owner,'apf-background')
    self.assertEqual(proof['bulk_object_count'],1024)
    payload.write_bytes(b'x'*((4<<20)+1))
    with self.assertRaisesRegex(RuntimeError,'output limit'):read_seed(owner,'apf-background')
if __name__=='__main__':unittest.main()
