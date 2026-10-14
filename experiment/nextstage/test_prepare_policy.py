import copy,unittest
from prepare_policy import desired,verify_delta,NAME
class Policy(unittest.TestCase):
 def test_exact_resolved_identities_and_positive_nonborrowed_capacity(self):
  objects=desired('kubeflow','actual-account');p,f=objects['items'];limited=p['spec']['limited']
  self.assertEqual(limited['nominalConcurrencyShares'],60);self.assertEqual(limited['lendablePercent'],0);self.assertEqual(limited['borrowingLimitPercent'],0)
  subjects=f['spec']['rules'][0]['subjects'];self.assertEqual(len(subjects),3);self.assertEqual(subjects[0]['serviceAccount']['name'],'actual-account');self.assertTrue(all(x['kind']=='ServiceAccount' for x in subjects))
  self.assertEqual(f['spec']['distinguisherMethod']['type'],'ByUser')
 def test_no_bootstrap_mutation_or_extra_policy_allowed(self):
  before={'flowschemas':{'items':[{'metadata':{'name':'stock'},'spec':{'x':1}}]},'priorities':{'items':[{'metadata':{'name':'stock'},'spec':{'x':2}}]}};after=copy.deepcopy(before);objects=desired('kubeflow','api')
  after['priorities']['items'].append(objects['items'][0]);after['flowschemas']['items'].append(objects['items'][1]);self.assertTrue(verify_delta(before,after,objects))
  after['priorities']['items'][0]['spec']['x']=3
  with self.assertRaises(RuntimeError):verify_delta(before,after,objects)
 def test_missing_resolved_account_refused(self):
  with self.assertRaises(RuntimeError):desired('kubeflow','')
