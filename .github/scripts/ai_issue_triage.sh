set -e
gh extension install github/gh-models --force || true

echo "Using model: $TARGET_MODEL"

read -r -d '' SYSTEM_INSTRUCTIONS << 'EOF'  || true
You are an expert open-source maintainer for Kubeflow.

Analyze the quality of the incoming issue based on Scope, Context, Guidance, and Complexity.
             
Calibrate your evaluation against these compressed reference standards:
- BACKEND (#13314): [backend] S3 operations fail with non-AWS object stores after AWS SDK v2 checksum defaults change. Clear isolated scope.
- BUG TIER (#13180): [bug] fix: E2E test flakiness on K8s v1.34.0 — root cause analysis. High-quality root-cause analysis and environment data.
- FRONTEND (#13108): [frontend] Adds coverage for frontend mock:api startup and enum drift. Explicit file paths and definitions of done.
- SDK (#12865): [sdk] [bug] [set_accelerator_limit] rejects valid accelerator counts (only allows 0, 1, 2, 4, 8, 16). Elite precision with failing parameters.

Compare the incoming issue detail density directly against the relevant blueprint standard above.

Respond strictly following this format structure without other markdown wraps:

### 📊 Scope
<Evaluate task boundaries and estimate the time window required to complete it>

### 📝 Context & Guidance
<Evaluate if steps, expected behavior, or links are provided against repo standards>

### ⚡ Complexity
<Rate difficulty: Low, Medium, High, detailing architectural deepness vs shallow tweaks>

### 🎯 Overall Issue Quality Verdict
<State clearly if this is ready for immediate pickup or if it requires more detail>
EOF

SAFE_BODY=$(printf "%s\n" "$RAW_BODY" | head -n 120)

USER_PAYLOAD="Title: ${TITLE}
Body: ${SAFE_BODY}"

export GH_MODELS_SYSTEM_PROMPT="$SYSTEM_INSTRUCTIONS"

if RESULT=$(printf "%s\n" "$USER_PAYLOAD" | gh models run "$TARGET_MODEL" 2>&1); then
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

EOF_DELIMITER=$(dd if=/dev/urandom bs=15 count=1 2>/dev/null | base64 | tr -dc 'a-zA-Z0-9')
echo "analysis<<$EOF_DELIMITER" >> $GITHUB_OUTPUT
echo "$ANALYSIS_REPORT" >> $GITHUB_OUTPUT
echo "$EOF_DELIMITER" >> $GITHUB_OUTPUT

          
echo "DEBUG: The raw analysis sent to output was:"
echo "$ANALYSIS_REPORT"