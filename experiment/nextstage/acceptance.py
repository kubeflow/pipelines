"""Strict evidence gates; absent or malformed data never passes."""
import math

def scheduled(rows, tenants):
    latest={row.get('tenant'):row for row in rows if 'complete' in row}
    return set(tenants)<=set(latest) and all(latest[t].get('complete') is True and latest[t].get('created') is True and latest[t].get('disabled') is True and latest[t].get('child_succeeded') is True for t in tenants)

def container_healthy(row):
    if row.get('reason')=='OOMKilled' or row.get('termination_reason')=='OOMKilled':return False
    if row.get('state')=='running':return row.get('ready') is True and row.get('pod_phase')=='Running'
    owners=row.get('controllers')
    completed_job=isinstance(owners,list) and len(owners)==1 and owners[0].get('kind')=='Job' and owners[0].get('api_version')=='batch/v1' and bool(owners[0].get('uid'))
    return completed_job and row.get('pod_phase')=='Succeeded' and row.get('state')=='terminated' and type(row.get('exit_code')) is int and row['exit_code']==0 and row.get('termination_reason')=='Completed'

def health(before,after):
    if not before or set(before)!=set(after):return False
    return all(before[k]['restart_count']==after[k]['restart_count'] and before[k].get('controllers')==after[k].get('controllers') and container_healthy(before[k]) and container_healthy(after[k]) for k in before)

def classification(records,policies,priority_uid,required_paths):
    """Records must be actual sanitized transport observations, never configured IDs."""
    matched=set()
    for r in records:
        if r.get('evidence_kind')!='response_headers':continue
        flow=policies.get(r.get('flow_schema_uid'))
        if not flow or flow.get('priority_uid')!=priority_uid or flow.get('priority_type')!='Limited':return False
        if r.get('priority_uid')!=priority_uid or r.get('status',0)<=0:return False
        matched.add(r.get('client_path'))
    return set(required_paths)<=matched

def recovery(samples,baseline):
    # Full first60 seconds, per caller. Baseline from quiet pre-pressure calibration.
    if not baseline:return False
    by={name:[] for name in baseline}
    for row in samples:
        if 0<=row.get('second',-1)<60 and row.get('caller') in by:by[row['caller']].append(row)
    for name,rows in by.items():
        if len(rows)!=60 or {r['second'] for r in rows}!=set(range(60)):return False
        base=baseline[name]
        if base.get('throughput',0)<=0 or not math.isfinite(base.get('p95_ms',float('nan'))):return False
        if any(r.get('missing') or r.get('queue',float('inf'))>base['queue'] for r in rows):return False
        # Window-percentiles cannot be averaged: require original bounded latency samples.
        latency=[x for r in rows for x in r.get('successful_latency_ms',[])]
        if not latency or len(latency)<.95*base['throughput']*60:return False
        latency.sort();p95=latency[min(len(latency)-1,math.ceil(.95*len(latency))-1)]
        if p95>1.2*base['p95_ms']:return False
    return True

def audit_classification(joined,expected_user,priority_uid):
    """Validate actual joined Kubernetes caller observations across configured clients."""
    if not joined.get('complete_for_captured_records') or joined.get('unmatched') or joined.get('ambiguous'):return False
    required={'tokenreviews','subjectaccessreviews','workflows','scheduledworkflows'}
    observed=set();core=False
    for row in joined.get('records',[]):
        if row.get('user')!=expected_user:continue
        if not row.get('audit_id') or not row.get('flow_uid') or row.get('priority_uid')!=priority_uid or row.get('priority_type')!='Limited':return False
        observed.add(row.get('resource'));core |= row.get('group')=='' and bool(row.get('resource'))
    return required<=observed and core
