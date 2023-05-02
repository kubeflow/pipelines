import argparse
import unittest

from commonv2.spec_input_parsers import SpecInputParsers as par


class SpecInputParsersTestCase(unittest.TestCase):
    def test_yaml_or_json_list(self):
        self.assertIsNone(par.yaml_or_json_list(""))
        self.assertEqual([], par.yaml_or_json_list("[ ]"))
        self.assertEqual(
            [{"a": 1}, {"b": 2}], par.yaml_or_json_list('[ {"a": 1}, {"b": 2} ]')
        )

        with self.assertRaises(argparse.ArgumentTypeError):
            par.yaml_or_json_list("{a: 1}")

        self.assertEqual(
            [{"a": 1}, {"b": 2}],
            par.yaml_or_json_list(
                """
        - a: 1 
        - b: 2"""
            ),
        )

    def test_yaml_or_json_dict(self):
        self.assertIsNone(par.yaml_or_json_dict(""))
        self.assertEqual({}, par.yaml_or_json_dict("{ }"))
        self.assertEqual({"a": 1}, par.yaml_or_json_dict('{"a": 1}'))

        with self.assertRaises(argparse.ArgumentTypeError):
            par.yaml_or_json_dict("[a: 1]")

        self.assertEqual(
            {"a": 1},
            par.yaml_or_json_dict(
                """
        a: 1"""
            ),
        )

    def test_str_to_bool(self):
        self.assertTrue(par.str_to_bool("True"))
        self.assertFalse(par.str_to_bool("False"))


if __name__ == "__main__":
    unittest.main()
