import json,pathlib,subprocess,sys,tempfile,unittest
HERE=pathlib.Path(__file__).parent
class Diagnostic(unittest.TestCase):
 def invoke(self,workers=64,rps=1000,proven=True,existing=False):
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d);(p/'run_pair.py').write_bytes((HERE/'run_pair.py').read_bytes())
   (p/'coordinator.py').write_text('''import json,sys,pathlib
 def_placeholder=0
'''.replace(' def_placeholder=0','')+'''def load(p,*args):return json.loads(pathlib.Path(p).read_text())
def save(p,v):pathlib.Path(p).write_text(json.dumps(v))
if __name__=='__main__':
 index=sys.argv[sys.argv.index('--index')+1]
 pathlib.Path(__file__).with_name('launched.json').write_text(json.dumps([index]))
 raise SystemExit(1)
''')
   (p/'calibration.json').write_text(json.dumps({'selected':{'workers':workers,'offered_rps':rps,'proven':proven},'measurements_authorized_by_calibration':True}))
   (p/'config.json').write_text(json.dumps({'background_calibration':str(p/'calibration.json')}))
   out=p/'out'
   if existing:(out/'arm-2').mkdir(parents=True)
   r=subprocess.run([sys.executable,str(p/'run_pair.py'),'--config',str(p/'config.json'),'--out',str(out)],capture_output=True)
   return r.returncode, json.loads((p/'launched.json').read_text()) if (p/'launched.json').exists() else None,[v.name for v in out.iterdir()]
 def test_only_negative_launched_failure_preserved(self):
  code,launched,files=self.invoke();self.assertEqual(code,1);self.assertEqual(launched,['2']);self.assertNotIn('arm-1-dispatch.json',files);self.assertIn('arm-2-outcome.json',files)
 def test_other_pressure_refused(self):
  for args in [{'workers':16},{'rps':256},{'proven':False}]:
   with self.subTest(args=args):self.assertIsNone(self.invoke(**args)[1])
 def test_existing_negative_never_repeated(self):self.assertIsNone(self.invoke(existing=True)[1])
 def test_calibration_only_prior_frozen_setting(self):
  import importlib.util
  spec=importlib.util.spec_from_file_location('scenario',HERE/'scenario.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
  self.assertEqual(m.STEPS,((64,1000),));self.assertEqual(m.SCENARIO['saturation_consecutive_samples'],10);self.assertEqual(m.SCENARIO['expected_shared_seats'],4)
if __name__=='__main__':unittest.main()
