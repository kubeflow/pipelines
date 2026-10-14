import unittest
from acceptance import scheduled,health,classification,recovery
class Gates(unittest.TestCase):
    def test_schedule_requires_every_tenant_actual_completion(self):
        row={'tenant':'a','created':True,'disabled':True,'child_succeeded':True,'complete':True}
        self.assertTrue(scheduled([row],['a']));self.assertFalse(scheduled([row],['a','b']))
        self.assertFalse(scheduled([dict(row,disabled=False)],['a']))
    def test_restart_and_replacement_fail(self):
        before={'uid/api':{'ready':True,'restart_count':0}}
        self.assertTrue(health(before,before));self.assertFalse(health(before,{'new/api':before['uid/api']}))
        self.assertFalse(health(before,{'uid/api':{'ready':True,'restart_count':1}}))
    def test_configured_classification_does_not_prove_observed(self):
        policies={'fs':{'priority_uid':'pl','priority_type':'Limited'}}
        row={'flow_schema_uid':'fs','priority_uid':'pl','client_path':'tokenreview','status':201}
        self.assertFalse(classification([row],policies,'pl',['tokenreview']))
        row['evidence_kind']='response_headers'
        self.assertTrue(classification([row],policies,'pl',['tokenreview']))
        self.assertFalse(classification([row],policies,'pl',['tokenreview','sar']))
    def test_recovery_needs_complete_raw_windows(self):
        baseline={'quiet':{'throughput':1,'p95_ms':10,'queue':0}}
        rows=[{'caller':'quiet','second':i,'queue':0,'successful_latency_ms':[10]} for i in range(60)]
        self.assertTrue(recovery(rows,baseline));self.assertFalse(recovery(rows[:-1],baseline))
        rows[30]['queue']=1;self.assertFalse(recovery(rows,baseline))
if __name__=='__main__':unittest.main()

class Audit(unittest.TestCase):
    def test_partial_join_and_missing_path_fail(self):
        from acceptance import audit_classification
        rows=[{'audit_id':str(i),'user':'api','resource':r,'group':'','flow_uid':'f','priority_uid':'p','priority_type':'Limited'} for i,r in enumerate(['tokenreviews','subjectaccessreviews','workflows','scheduledworkflows','pods'])]
        joined={'complete_for_captured_records':True,'records':rows,'unmatched':0,'ambiguous':0}
        self.assertTrue(audit_classification(joined,'api','p'))
        self.assertFalse(audit_classification(dict(joined,ambiguous=1),'api','p'))
        self.assertFalse(audit_classification(dict(joined,records=rows[:2]),'api','p'))
