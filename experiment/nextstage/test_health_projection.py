import json,subprocess,tempfile,unittest
from pathlib import Path
from integration import HEALTH_TEMPLATE,health_snapshot
from acceptance import health
from coordinator import safe_failure_reason
RENDER='''package main
import("encoding/json";"os";"text/template")
func main(){var data interface{};f,e:=os.Open(os.Args[1]);if e!=nil{panic(e)};defer f.Close();if e=json.NewDecoder(f).Decode(&data);e!=nil{panic(e)};t,e:=template.New("health").Parse(os.Args[2]);if e!=nil{panic(e)};if e=t.Execute(os.Stdout,data);e!=nil{panic(e)}}
'''
class Projection(unittest.TestCase):
 def test_real_go_template_large_spec_and_same_health_fields(self):
  def pod(ns,uid,ready=True):return {'metadata':{'namespace':ns,'uid':uid},'spec':{'args':['private'*350000]},'status':{'phase':'Running','containerStatuses':[{'state':{'running':{'startedAt':'now'}},'name':'api','restartCount':2,'ready':ready,'lastState':{'terminated':{'reason':'OOMKilled'}}},{'state':{'running':{'startedAt':'now'}},'name':'sidecar','restartCount':0,'ready':True,'lastState':{}}]}}
  obj={'items':[pod('kubeflow','kfp',False),pod('kube-system','system'),pod('tenant','excluded')]}
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);(root/'render.go').write_text(RENDER);(root/'pods.json').write_text(json.dumps(obj))
   self.assertGreater((root/'pods.json').stat().st_size,1<<20)
   rendered=subprocess.run(['go','run',str(root/'render.go'),str(root/'pods.json'),HEALTH_TEMPLATE],check=True,capture_output=True).stdout
  self.assertLess(len(rendered),2000);self.assertNotIn(b'private',rendered)
  class Owner:
   def k(self,*args,**kwargs):
    assert args[:5]==('get','pods','-A','-o','go-template');return rendered
  result=health_snapshot(Owner())
  self.assertEqual(set(result),{'kfp/api','kfp/sidecar','system/api','system/sidecar'})
  self.assertEqual(result['kfp/api'],{'ready':False,'restart_count':2,'reason':'OOMKilled','state':'running','pod_phase':'Running','controllers':[],'exit_code':None,'termination_reason':None})
  self.assertFalse(health(result,result))
 def test_real_job_termination_projection(self):
  obj={'items':[{'metadata':{'namespace':'kubeflow','uid':'pod','ownerReferences':[{'apiVersion':'batch/v1','kind':'Job','uid':'controller','controller':True}]},'status':{'phase':'Succeeded','containerStatuses':[{'name':'unrelated-name','ready':False,'restartCount':0,'lastState':{},'state':{'terminated':{'exitCode':0,'reason':'Completed'}}}]}}]}
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);(root/'render.go').write_text(RENDER);(root/'pods.json').write_text(json.dumps(obj))
   rendered=subprocess.run(['go','run',str(root/'render.go'),str(root/'pods.json'),HEALTH_TEMPLATE],check=True,capture_output=True).stdout
  class Owner:
   def k(self,*a,**kw):return rendered
  result=health_snapshot(Owner());self.assertTrue(health(result,result));row=result['pod/unrelated-name']
  self.assertEqual(row['controllers'],[{'api_version':'batch/v1','kind':'Job','uid':'controller'}]);self.assertEqual(row['exit_code'],0)
 def test_safe_reason_never_exposes_arbitrary_output(self):
  self.assertEqual(safe_failure_reason(RuntimeError('bounded command output limit')),'bounded command output limit')
  self.assertEqual(safe_failure_reason(RuntimeError('Bearer secret')),'detail_suppressed')
if __name__=='__main__':unittest.main()
