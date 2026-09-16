from __future__ import annotations

import importlib.util
import pathlib
import sys
import unittest

MODULE_PATH = pathlib.Path(__file__).with_name("contributor-report.py")
SPEC = importlib.util.spec_from_file_location("contributor_report", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class ContributorReportTest(unittest.TestCase):

    def test_parse_kubeflow_org_members_includes_admins_and_members(self):
        members = MODULE.parse_kubeflow_org_members("""
orgs:
    kubeflow:
        admins:
        - AdminUser
        members:
        - MemberUser
        billing_email: test@example.com
        teams:
          team-a:
            members:
            - TeamUser
""")
        self.assertEqual(members, {"adminuser", "memberuser"})

    def test_non_user_rows_report_not_applicable(self):
        rows = MODULE.build_non_user_rows("Bot")
        self.assertEqual(rows[0].metric, "GitHub author type")
        self.assertEqual(rows[0].value, "Bot")
        self.assertTrue(all(row.value == "N/A" for row in rows[1:]))

    def test_build_comment_renders_markdown_table(self):
        rows = [
            MODULE.MarkdownRow(metric="Metric A", value="1", notes="Note A"),
            MODULE.MarkdownRow(metric="Metric B", value="2", notes="Note B"),
        ]
        comment = MODULE.build_comment("alice", rows)
        self.assertIn("| Metric | Value | Notes |", comment)
        self.assertIn("| Metric A | 1 | Note A |", comment)
        self.assertIn("| Metric B | 2 | Note B |", comment)

    def test_list_issue_comments_paginates_until_short_page(self):
        calls = []
        pages = {
            1: [{
                "id": i
            } for i in range(100)],
            2: [{
                "id": 100
            }, {
                "id": 101
            }],
        }

        def fake_request(path, method="GET", body=None):
            del method, body
            calls.append(path)
            page = int(path.split("page=")[-1])
            return pages[page]

        original = MODULE.github_request
        try:
            MODULE.github_request = fake_request
            comments = MODULE.list_issue_comments("kubeflow", "pipelines", 123)
        finally:
            MODULE.github_request = original

        self.assertEqual(len(comments), 102)
        self.assertEqual(len(calls), 2)
        self.assertTrue(calls[0].endswith("page=1"))
        self.assertTrue(calls[1].endswith("page=2"))

    def test_is_human_user_checks_github_type(self):
        self.assertTrue(MODULE.is_human_user({"type": "User"}))
        self.assertFalse(MODULE.is_human_user({"type": "Bot"}))
        self.assertFalse(MODULE.is_human_user({"type": "Organization"}))


if __name__ == "__main__":
    unittest.main()
