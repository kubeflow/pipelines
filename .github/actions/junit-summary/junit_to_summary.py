#!/usr/bin/env python3
"""
Parse JUnit XML reports and generate GitHub step summary with optional custom data.
"""

import argparse
import glob
import json
import os
import sys
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional


@dataclass
class TestCase:
    """Represents a single test case."""
    name: str
    classname: str
    time: float
    status: str  # passed, failed, skipped, error
    failure_message: Optional[str] = None
    failure_type: Optional[str] = None
    error_message: Optional[str] = None
    system_out: Optional[str] = None
    system_err: Optional[str] = None


@dataclass
class TestSuite:
    """Represents a test suite."""
    name: str
    tests: int
    failures: int
    errors: int
    skipped: int
    time: float
    test_cases: List[TestCase]


@dataclass
class TestReport:
    """Represents the complete test report."""
    total_tests: int
    total_failures: int
    total_errors: int
    total_skipped: int
    total_passed: int
    total_time: float
    test_suites: List[TestSuite]


def parse_junit_xml(xml_file: Path) -> TestSuite:
    """Parse a JUnit XML file and extract test results."""
    tree = ET.parse(xml_file)
    root = tree.getroot()

    # Handle both <testsuites> and <testsuite> as root
    if root.tag == 'testsuites':
        # Multiple test suites
        suites = []
        for suite_elem in root.findall('testsuite'):
            suite = parse_test_suite(suite_elem)
            suites.append(suite)

        # Aggregate all suites
        total_tests = sum(s.tests for s in suites)
        total_failures = sum(s.failures for s in suites)
        total_errors = sum(s.errors for s in suites)
        total_skipped = sum(s.skipped for s in suites)
        total_time = sum(s.time for s in suites)
        all_test_cases = [tc for s in suites for tc in s.test_cases]

        return TestSuite(
            name="All Suites",
            tests=total_tests,
            failures=total_failures,
            errors=total_errors,
            skipped=total_skipped,
            time=total_time,
            test_cases=all_test_cases
        )
    else:
        # Single test suite
        return parse_test_suite(root)


def parse_test_suite(suite_elem: ET.Element) -> TestSuite:
    """Parse a single testsuite element."""
    name = suite_elem.get('name', 'Unknown')
    tests = int(suite_elem.get('tests', 0))
    failures = int(suite_elem.get('failures', 0))
    errors = int(suite_elem.get('errors', 0))
    skipped = int(suite_elem.get('skipped', 0))
    time = float(suite_elem.get('time', 0))

    test_cases = []
    for testcase_elem in suite_elem.findall('testcase'):
        test_case = parse_test_case(testcase_elem)
        test_cases.append(test_case)

    return TestSuite(
        name=name,
        tests=tests,
        failures=failures,
        errors=errors,
        skipped=skipped,
        time=time,
        test_cases=test_cases
    )


def parse_test_case(testcase_elem: ET.Element) -> TestCase:
    """Parse a single testcase element."""
    name = testcase_elem.get('name', 'Unknown')
    classname = testcase_elem.get('classname', '')
    time = float(testcase_elem.get('time', 0))
    status = testcase_elem.get('status', 'passed')

    # Check for failure
    failure_elem = testcase_elem.find('failure')
    failure_message = None
    failure_type = None
    if failure_elem is not None:
        status = 'failed'
        failure_message = failure_elem.get('message', '')
        failure_type = failure_elem.get('type', '')

    # Check for error
    error_elem = testcase_elem.find('error')
    error_message = None
    if error_elem is not None:
        status = 'error'
        error_message = error_elem.get('message', '')

    # Check for skipped
    skipped_elem = testcase_elem.find('skipped')
    if skipped_elem is not None:
        status = 'skipped'

    # Get system output
    system_out_elem = testcase_elem.find('system-out')
    system_out = system_out_elem.text if system_out_elem is not None else None

    system_err_elem = testcase_elem.find('system-err')
    system_err = system_err_elem.text if system_err_elem is not None else None

    return TestCase(
        name=name,
        classname=classname,
        time=time,
        status=status,
        failure_message=failure_message,
        failure_type=failure_type,
        error_message=error_message,
        system_out=system_out,
        system_err=system_err
    )


def aggregate_reports(xml_files: List[Path]) -> TestReport:
    """Aggregate multiple JUnit XML files into a single report."""
    all_suites = []

    for xml_file in xml_files:
        try:
            suite = parse_junit_xml(xml_file)
            all_suites.append(suite)
        except Exception as e:
            print(f"Error parsing {xml_file}: {e}", file=sys.stderr)
            continue

    total_tests = sum(s.tests for s in all_suites)
    total_failures = sum(s.failures for s in all_suites)
    total_errors = sum(s.errors for s in all_suites)
    total_skipped = sum(s.skipped for s in all_suites)
    total_passed = total_tests - total_failures - total_errors - total_skipped
    total_time = sum(s.time for s in all_suites)

    return TestReport(
        total_tests=total_tests,
        total_failures=total_failures,
        total_errors=total_errors,
        total_skipped=total_skipped,
        total_passed=total_passed,
        total_time=total_time,
        test_suites=all_suites
    )


def format_duration(seconds: float) -> str:
    """Format duration in seconds to a human-readable string."""
    if seconds < 60:
        return f"{seconds:.2f}s"
    elif seconds < 3600:
        minutes = int(seconds // 60)
        secs = seconds % 60
        return f"{minutes}m {secs:.2f}s"
    else:
        hours = int(seconds // 3600)
        minutes = int((seconds % 3600) // 60)
        secs = seconds % 60
        return f"{hours}h {minutes}m {secs:.2f}s"


def generate_markdown_summary(report: TestReport, custom_data: Optional[Dict] = None) -> str:
    """Generate a markdown summary from the test report."""
    md_lines = []

    # Add custom data section if provided
    if custom_data:
        for key, value in custom_data.items():
            md_lines.append(f"**{key}**: {value}\n")
        md_lines.append("")

    # Overall results
    success_rate = (report.total_passed / report.total_tests * 100) if report.total_tests > 0 else 0

    # Status emoji
    if report.total_failures == 0 and report.total_errors == 0:
        status_emoji = "✅"
        status_text = "PASSED"
    else:
        status_emoji = "❌"
        status_text = "FAILED"

    md_lines.append(f"## {status_emoji} Overall Status: {status_text}\n")

    # Summary table
    md_lines.append("| Total Tests | ✅ Passed | ❌ Failed | ⚠️ Errors | ⏭️ Skipped | ⏱️ Duration | 📊 Success Rate |")
    md_lines.append("|-------------|-----------|-----------|-----------|------------|-------------|-----------------|")
    md_lines.append(f"| {report.total_tests} | {report.total_passed} | {report.total_failures} | {report.total_errors} | {report.total_skipped} | {format_duration(report.total_time)} | {success_rate:.2f}% |")
    md_lines.append("")

    # Failed tests details
    if report.total_failures > 0 or report.total_errors > 0:
        md_lines.append("## ❌ Failed Tests\n")

        failed_tests = []
        for suite in report.test_suites:
            for test_case in suite.test_cases:
                if test_case.status in ['failed', 'error']:
                    failed_tests.append((suite.name, test_case))

        if failed_tests:
            md_lines.append("| Test Name | Suite | Duration | Error |")
            md_lines.append("|-----------|-------|----------|-------|")

            for suite_name, test_case in failed_tests:
                # Truncate long test names
                test_name = test_case.name[:80] + "..." if len(test_case.name) > 80 else test_case.name
                error_msg = test_case.failure_message or test_case.error_message or "No message"
                error_msg = error_msg[:100] + "..." if len(error_msg) > 100 else error_msg
                # Escape pipe characters in error message
                error_msg = error_msg.replace("|", "\\|")

                md_lines.append(f"| {test_name} | {suite_name} | {test_case.time:.2f}s | {error_msg} |")

            md_lines.append("")

    # Test suites breakdown
    if len(report.test_suites) > 1:
        md_lines.append("## 📋 Test Suites Breakdown\n")
        md_lines.append("| Suite Name | Tests | Passed | Failed | Errors | Skipped | Duration |")
        md_lines.append("|------------|-------|--------|--------|--------|---------|----------|")

        for suite in report.test_suites:
            passed = suite.tests - suite.failures - suite.errors - suite.skipped
            suite_name = suite.name[:50] + "..." if len(suite.name) > 50 else suite.name
            md_lines.append(
                f"| {suite_name} | {suite.tests} | {passed} | {suite.failures} | "
                f"{suite.errors} | {suite.skipped} | {format_duration(suite.time)} |"
            )

        md_lines.append("")

    return "\n".join(md_lines)


def expand_file_patterns(patterns: List[str]) -> List[Path]:
    """Expand glob patterns and collect all XML files."""
    xml_files = []

    for pattern in patterns:
        pattern_path = Path(pattern)

        if pattern_path.is_dir():
            # If it's a directory, find all XML files in it
            found_files = list(pattern_path.rglob('*.xml'))
        elif pattern_path.is_file():
            # If it's a file, use it directly
            found_files = [pattern_path]
        else:
            # Try to expand as a glob pattern
            found_files = list(Path('.').glob(pattern))

        xml_files.extend(found_files)

    if not xml_files:
        print(f"Error: No XML files found matching the patterns: {patterns}", file=sys.stderr)
        sys.exit(1)

    print(f"Found XML files: {[str(f) for f in xml_files]}")
    return xml_files


def set_github_output(key: str, value: str):
    """Set a GitHub Action output."""
    github_output = os.environ.get('GITHUB_OUTPUT')

    if github_output:
        with open(github_output, 'a') as f:
            f.write(f"{key}={value}\n")
    else:
        print(f"Would set output: {key}={value}")


def set_github_outputs(report: TestReport):
    """Set all GitHub Action outputs based on the test report."""
    success_rate = (report.total_passed / report.total_tests * 100) if report.total_tests > 0 else 0

    set_github_output("total_tests", str(report.total_tests))
    set_github_output("total_passed", str(report.total_passed))
    set_github_output("total_failed", str(report.total_failures))
    set_github_output("total_errors", str(report.total_errors))
    set_github_output("total_skipped", str(report.total_skipped))
    set_github_output("success_rate", f"{success_rate:.2f}")


def write_to_step_summary(markdown: str):
    """Write markdown to GitHub step summary."""
    github_step_summary = os.environ.get('GITHUB_STEP_SUMMARY')

    if not github_step_summary:
        print("Warning: GITHUB_STEP_SUMMARY environment variable not set", file=sys.stderr)
        print("Markdown output:")
        print(markdown)
        return

    with open(github_step_summary, 'a') as f:
        f.write(markdown)
        f.write("\n")

    print(f"Successfully wrote summary to {github_step_summary}")


def main():
    parser = argparse.ArgumentParser(
        description='Parse JUnit XML reports and generate GitHub step summary'
    )
    parser.add_argument(
        'xml_files',
        nargs='+',
        help='Path(s) to JUnit XML file(s) or directories (supports glob patterns)'
    )
    parser.add_argument(
        '--custom-data',
        type=str,
        help='JSON string with custom data to include in the summary'
    )
    parser.add_argument(
        '--custom-data-file',
        type=Path,
        help='Path to JSON file with custom data to include in the summary'
    )
    parser.add_argument(
        '--fail-on-test-failures',
        action='store_true',
        default=True,
        help='Whether to fail the step if tests fail (default: True)'
    )
    parser.add_argument(
        '--no-fail-on-test-failures',
        dest='fail_on_test_failures',
        action='store_false',
        help='Do not fail the step if tests fail'
    )
    parser.add_argument(
        '--output',
        type=Path,
        help='Output file for markdown (default: write to GITHUB_STEP_SUMMARY)'
    )

    args = parser.parse_args()

    # Expand file patterns and collect XML files
    xml_files = expand_file_patterns(args.xml_files)

    # Parse custom data
    custom_data = None
    if args.custom_data:
        try:
            custom_data = json.loads(args.custom_data)
        except json.JSONDecodeError as e:
            print(f"Error parsing custom data JSON: {e}", file=sys.stderr)
            sys.exit(1)
    elif args.custom_data_file:
        try:
            with open(args.custom_data_file) as f:
                custom_data = json.load(f)
        except (FileNotFoundError, json.JSONDecodeError) as e:
            print(f"Error reading custom data file: {e}", file=sys.stderr)
            sys.exit(1)

    # Parse and aggregate reports
    report = aggregate_reports(xml_files)

    # Set GitHub Action outputs
    set_github_outputs(report)

    # Generate markdown
    markdown = generate_markdown_summary(report, custom_data)

    # Write output
    if args.output:
        with open(args.output, 'w') as f:
            f.write(markdown)
        print(f"Wrote markdown summary to {args.output}")
    else:
        write_to_step_summary(markdown)

    # Exit with failure if tests failed and fail_on_test_failures is True
    if args.fail_on_test_failures and (report.total_failures > 0 or report.total_errors > 0):
        sys.exit(1)


if __name__ == '__main__':
    main()