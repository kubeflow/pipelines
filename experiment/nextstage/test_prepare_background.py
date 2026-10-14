import unittest
from prepare_background import resources,classification
class BackgroundTests(unittest.TestCase):
 def test_role_is_namespace_scoped_named_configmap_only(self):
  objects=resources();role=next(o for o in objects if o['kind']=='Role')
  self.assertEqual(role['rules'],[{'apiGroups':[''],'resources':['configmaps'],'resourceNames':['probe'],'verbs':['get','list','update']}])
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
if __name__=='__main__':unittest.main()
