import json, unittest
from diagnose_import import project
class ProjectionTest(unittest.TestCase):
    def test_config_is_bounded_to_hashes_and_rootfs(self):
        raw=json.dumps({'architecture':'amd64','os':'linux','rootfs':{'diff_ids':['sha256:'+'a'*64]},'config':{'Env':['SECRET=hidden-value'],'Cmd':['sensitive']},'history':[{'created_by':'hidden-history'}]}).encode()
        p=project(raw)
        self.assertEqual(p['kind'],'config')
        self.assertEqual(p['diff_ids'],['sha256:'+'a'*64])
        self.assertNotIn('hidden',json.dumps(p));self.assertNotIn('sensitive',json.dumps(p))
        self.assertIn('Env',p['runtime_fields_sha256'])
    def test_manifest_only_keeps_digest_children(self):
        p=project(json.dumps({'manifests':[{'digest':'sha256:'+'a'*64,'annotations':{'private':'hidden'}},{'digest':'invalid'}]}).encode())
        self.assertEqual(p['children'],['sha256:'+'a'*64]);self.assertNotIn('hidden',json.dumps(p))
if __name__=='__main__':unittest.main()
