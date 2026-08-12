import os
import re
import sys
import time
import subprocess

# Configuration from Environment Variables
TARGET_MODEL = os.getenv("TARGET_MODEL", "openai/gpt-4o-mini")
ISSUE_TITLE = os.getenv("TITLE", "")
RAW_BODY = os.getenv("RAW_BODY", "")
GH_TOKEN = os.getenv("GH_TOKEN")
ISSUE_NUMBER = os.getenv("ISSUE_NUMBER")

def run_model_with_retry(system_prompt, user_prompt, max_retries=3):
    """Executes gh models run with exponential backoff retry logic."""
    delay = 2
    for attempt in range(1, max_retries + 1):
        cmd = ["gh", "models", "run", TARGET_MODEL, "--system-prompt", system_prompt, user_prompt]
        result = subprocess.run(cmd, capture_output=True, text=True)
        
        if result.returncode == 0:
            return result.stdout.strip()
        
        print(f"⚠️ Attempt {attempt} failed. Retrying in {delay}s... Error: {result.stderr.strip()}")
        time.sleep(delay)
        delay *= 2  # Exponential backoff
        
    return None


def validate_and_parse_title(title):
    """Validates title format: <type>(<area>): <content>"""
    pattern = r"^([a-z]+)\(([a-z]+)\):\s*(.+)$"
    match = re.match(pattern, title.strip())
    if not match:
        return None, None, None
    return match.group(1).lower(), match.group(2).lower(), match.group(3)

def main():
    print(f"Using model: {TARGET_MODEL}")

    # Step 1: Validate Title Format
    issue_type, issue_area, issue_content = validate_and_parse_title(ISSUE_TITLE)
    if not issue_type or issue_type not in ["bug", "chore", "feature"]:
        error_msg = (
            "🤖 **AI Issue Quality Review**\n\n"
            "⚠️ **Validation Failed:** Issue title must follow the correct format: "
            "`<type>(<area>): <title contents>` where type is `bug`, `chore`, or `feature`."
        )
        print("❌ Title format invalid.")
        github_output_path = os.getenv("GITHUB_OUTPUT")
        if github_output_path:
            with open(github_output_path, "a") as f:
                f.write("analysis<<EOF\n")
                f.write(error_msg + "\n")
                f.write("EOF\n")
        sys.exit(0)

    print(f"✅ Title validated successfully. Type: {issue_type}, Area: {issue_area}")

    system_instructions = """You are an expert open-source maintainer for Kubeflow Pipelines.
You are an expert open-source maintainer for Kubeflow Pipelines.
Analyze the quality of the incoming issue {issue_type} based on Scope, Context, Guidance, and Complexity.
             
Calibrate your evaluation against these compressed reference standards:
- BACKEND (#13314): [backend] S3 operations fail with non-AWS object stores after AWS SDK v2 checksum defaults change. Clear isolated scope.
- BUG TIER (#13180): [bug] fix: E2E test flakiness on K8s v1.34.0 — root cause analysis. High-quality root-cause analysis and environment data.
- FRONTEND (#13108): [frontend] Adds coverage for frontend mock:api startup and enum drift. Explicit file paths and definitions of done.
- SDK (#12865): [sdk] [bug] [set_accelerator_limit] rejects valid accelerator counts. Elite precision with failing parameters.

Respond strictly following this format structure without other markdown wraps:
- Each section MUST contain exactly 2 to 3 short, bullet fragments. 
- Do NOT write full-length paragraphs or introductory text. Keep it highly concise.
- Do NOT include any time frame or implementation window estimations.


### 📊 Scope
- <If the technical task boundaries are clear or ambiguous>
- <If the issue isolates specific components, files, or packages correctly>

### 📝 Context & Guidance
<Evaluate if steps, expected behavior, or links are provided against repo standards>

### ⚡ Complexity
- <State difficulty tier: Low, Medium, or High>
- <Break down breadth and depth of the proposed change>

### 🎯 Overall Issue Quality Verdict
- <State definitively if this is ready for immediate developer pickup>
- <Outline the single most impactful recommendation>
"""

    user_prompt = f"Title: {ISSUE_TITLE} | Body: {RAW_BODY}"

    analysis_report = run_model_with_retries = run_model_with_retry(system_instructions, user_prompt)
    
    if not analysis_report:
        print("⚠️ CRITICAL: AI Model execution failed after multiple retries.")
        analysis_report = (
            "### ⚠️ Automated Triage Skipped\n"
            "The issue body text or environment logs exceeded processing size boundaries or API limits for this triage pass."
        )
    
    # Set GitHub Actions output
    github_output_path = os.getenv("GITHUB_OUTPUT")
    if github_output_path:
        with open(github_output_path, "a") as f:
            f.write("analysis<<EOF\n")
            f.write(analysis_report + "\n")
            f.write("EOF\n")

    print("DEBUG: The raw analysis sent to output was:")
    print(analysis_report)

if __name__ == "__main__":
    main()