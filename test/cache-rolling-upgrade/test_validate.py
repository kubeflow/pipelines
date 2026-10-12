"""Regression checks for retained legacy completion acceptance."""
import unittest
from unittest import mock

import validate


class RetainedCompletionTest(unittest.TestCase):

    def test_waits_for_explicit_rejection(self):
        rejection = (
            'skipping cache write for pod cache-a/old-inflight: '
            'execution cache key does not match namespace and template')
        with mock.patch.object(validate, 'k', side_effect=['', rejection]) as logs, \
             mock.patch.object(validate.time, 'sleep'), \
             mock.patch.object(validate, 'get', return_value={'metadata': {'labels': {}}}), \
             mock.patch.object(validate, 'sql', return_value='0'):
            validate.wait_for_legacy_completion_rejection()
        self.assertEqual(logs.call_count, 2)

    def test_missing_rejection_cannot_pass(self):
        with mock.patch.object(validate, 'k', return_value='not eligible'), \
             mock.patch.object(validate.time, 'monotonic', side_effect=[0, 0, 121]), \
             mock.patch.object(validate.time, 'sleep'):
            with self.assertRaisesRegex(AssertionError, 'Timed out'):
                validate.wait_for_legacy_completion_rejection()

    def test_rechecks_late_label_and_database_writes(self):
        for labels, count in (({validate.CACHE_ID: '12'}, '0'), ({}, '1')):
            with self.subTest(labels=labels, count=count), \
                 mock.patch.object(validate, 'get', return_value={'metadata': {'labels': labels}}), \
                 mock.patch.object(validate, 'sql', return_value=count):
                with self.assertRaises(AssertionError):
                    validate.assert_legacy_completion_unwritten()


if __name__ == '__main__':
    unittest.main()
