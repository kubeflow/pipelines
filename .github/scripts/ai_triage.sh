set -e
gh extension install github/gh-models --force || true

echo "Using model: $TARGET_MODEL"

read -r -d '' SYSTEM_INSTRUCTIONS << 'EOF'  || true
You are an expert open-source maintainer for Kubeflow Pipelines.

Analyze the quality of the incoming issue based on Scope, Context, Guidance, and Complexity.
             
Calibrate your evaluation against these compressed reference standards:
- BACKEND (#13314): [backend] S3 operations fail with non-AWS object stores after AWS SDK v2 checksum defaults change. Clear isolated scope.
- BUG TIER (#13180): [bug] fix: E2E test flakiness on K8s v1.34.0 — root cause analysis. High-quality root-cause analysis and environment data.
- FRONTEND (#13108): [frontend] Adds coverage for frontend mock:api startup and enum drift. Explicit file paths and definitions of done.
- SDK (#12865): [sdk] [bug] [set_accelerator_limit] rejects valid accelerator counts (only allows 0, 1, 2, 4, 8, 16). Elite precision with failing parameters.

Compare the incoming issue detail density directly against the relevant blueprint standard above.

Respond strictly following this format structure without other markdown wraps:

- Each section MUST contain exactly 2 to 3 short, bullet fragments. 
- Do NOT write full-length paragraphs or introductory text. Keep it highly concise for quick scanning.
- Do NOT include any time frame or implementation window estimations.


### 📊 Scope
- <If the technical task boundaries are clear or ambiguous>
- <If the issue isolates specific components, files, or packages correctly>


### 📝 Context & Guidance
<Evaluate if steps, expected behavior, or links are provided against repo standards>

### ⚡ Complexity
- <State difficulty tier: Low, Medium, or High, calibrated against this exact rubric:
  * LOW: Task is isolated to single-file fixes, shallow tweaks, or documentation updates.
  * MEDIUM: Task has moderate architectural depth, affecting internal logic patterns or specific layer wrappers.
  * HIGH: Task has deep architectural depth or high breadth, spanning multiple system components simultaneously (e.g., changes across the SDK, backend, frontend, or argo compiler engines).>
- <Break down the breadth (cross-layer impact) and depth (internal system complexity) of the proposed change>


### 🎯 Overall Issue Quality Verdict
- <State definitively if this is ready for immediate developer pickup>
- <Outline the single most impactful recommendation to improve the issue quality>
EOF


USER_PROMPT="Title: ${TITLE} | Body: ${RAW_BODY}"

if RESULT=$(gh models run "$TARGET_MODEL" --system-prompt "$SYSTEM_INSTRUCTIONS" "$USER_PROMPT" 2>&1); then
    echo "✅ AI model executed successfully."
    ANALYSIS_REPORT="$RESULT"
else
    echo "⚠️ CRITICAL: AI Model execution failed or preview tier limit hit."
    echo "Error details: $RESULT"
            
    read -r -d '' ANALYSIS_REPORT << 'EOF' || true
    ### ⚠️ Automated Triage Skipped
    The issue body text or environment logs exceeded processing size boundaries for this triage pass.
EOF
fi

echo "analysis<<EOF" >> $GITHUB_OUTPUT
echo "$ANALYSIS_REPORT" >> $GITHUB_OUTPUT
echo "EOF" >> $GITHUB_OUTPUT

          
echo "DEBUG: The raw analysis sent to output was:"
echo "$ANALYSIS_REPORT"
