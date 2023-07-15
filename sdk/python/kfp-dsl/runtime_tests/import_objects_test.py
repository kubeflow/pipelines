class TestImportObjects:

    def test(self):
        # from kfp.dsl import * only allowed at module level, so emulate behavior
        from kfp import dsl
        for obj_name in dir(dsl):
            if not obj_name.startswith('_'):
                getattr(dsl, obj_name)
