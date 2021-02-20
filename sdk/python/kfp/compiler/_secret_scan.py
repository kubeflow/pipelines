#!/bin/python3

import json
import sys
import re
import string
import warnings
import yaml


# Taken directly from TruffleHog
# https://github.com/dxa4481/truffleHog/blob/0f223225d6efc8c64504d9381eececb06b14c0e6/truffleHog/truffleHog.py#L120-L138
# Therefore GPL Licensed

import math

BASE64_CHARS = set("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=")
HEX_CHARS = set("1234567890abcdefABCDEF")
ASCII_CHARS = set(string.printable)
MAX_ENTROPY = 3.5

SAFE_ENDINGS = [
    'apiVersion',   # High entropy
    'name',         # Env vars
    'image',        # Images and registries have high entropy
    'description',  # Argo specific, long human description
    'template',     # Argo specific, references parts of the DAG by name
    'entrypoint',
    'pipelines.kubeflow.org/pipeline_compilation_time'
]
SAFE_ENDINGS = [x.lower() for x in SAFE_ENDINGS]

def shannon_entropy(data, iterator):
    """
    Borrowed from http://blog.dkbza.org/2007/05/scanning-data-for-entropy-anomalies.html
    """
    if not data:
        return 0
    entropy = 0
    for x in iterator:
        p_x = float(data.count(x))/len(data)
        if p_x > 0:
            entropy += - p_x*math.log(p_x, 2)
    return entropy




# Borrowed from @wg102
# https://github.com/wg102/kubeflow_pipeline_detection

def seq_iter(obj):
    if isinstance(obj, dict):
        return obj.items()
    elif isinstance(obj, list):
        return enumerate(obj)
    else:
        print("WARNING: obj is neither a dict not a list. Skipping", file=sys.stderr)
        return []

def traversal(tree, parent=[]):
    """
    Get all (key, value) pairs in the object tree, where the "key"
    is the entire path from the root

    Args:
        tree: A recursive dict object (a parsed yaml file)

    Returns:
        An iterator of all (path, leaf) tuples. I.e. every
        key/value in the yaml file.
    """
    for (k, v) in seq_iter(tree):
        if k == 'pipelines.kubeflow.org/pipeline_spec' and isinstance(v, str):
            # This is a special case where the annotation has a json string
            try:
                yield from traversal(json.loads(v), parent=parent+[k])
            except:
                yield (parent + [k], v)
        elif any((isinstance(v, t) for t in (str, int, bool, float))):
            yield (parent + [k], v)
        else:
            yield from traversal(v, parent=parent+[k])



# Taken from TruffleHog
# https://raw.githubusercontent.com/feeltheajf/trufflehog3/master/truffleHog3/rules.yaml
# GPL Licensed
rules = {
    "Slack Token": "(xox[p|b|o|a]-[0-9]{12}-[0-9]{12}-[0-9]{12}-[a-z0-9]{32})"
    ,"RSA private key": "-----BEGIN RSA PRIVATE KEY-----"
    ,"SSH (DSA) private key": "-----BEGIN DSA PRIVATE KEY-----"
    ,"SSH (EC) private key": "-----BEGIN EC PRIVATE KEY-----"
    ,"PGP private key block": "-----BEGIN PGP PRIVATE KEY BLOCK-----"
    ,"Amazon AWS Access Key ID": "AKIA[0-9A-Z]{16}"
    ,"Amazon MWS Auth Token": "amzn\\.mws\\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
    ,"AWS API Key": "AKIA[0-9A-Z]{16}"
    ,"Facebook Access Token": "EAACEdEose0cBA[0-9A-Za-z]+"
    ,"Facebook OAuth": '[f|F][a|A][c|C][e|E][b|B][o|O][o|O][k|K].*[''|"][0-9a-f]{32}[''|"]'
    ,"GitHub": '[g|G][i|I][t|T][h|H][u|U][b|B].*[''|"][0-9a-zA-Z]{35,40}[''|"]'
    ,"Generic API Key": '[a|A][p|P][i|I][_]?[k|K][e|E][y|Y].*[''|"][0-9a-zA-Z]{32,45}[''|"]'
    ,"Generic Secret": '[s|S][e|E][c|C][r|R][e|E][t|T].*[''|"][0-9a-zA-Z]{32,45}[''|"]'
    ,"Google API Key": "AIza[0-9A-Za-z\\-_]{35}"
    ,"Google Cloud Platform API Key": "AIza[0-9A-Za-z\\-_]{35}"
    ,"Google Cloud Platform OAuth": "[0-9]+-[0-9A-Za-z_]{32}\\.apps\\.googleusercontent\\.com"
    ,"Google Drive API Key": "AIza[0-9A-Za-z\\-_]{35}"
    ,"Google Drive OAuth": "[0-9]+-[0-9A-Za-z_]{32}\\.apps\\.googleusercontent\\.com"
    ,"Google (GCP) Service-account": '"type: "service_account"'
    ,"Google Gmail API Key": "AIza[0-9A-Za-z\\-_]{35}"
    ,"Google Gmail OAuth": "[0-9]+-[0-9A-Za-z_]{32}\\.apps\\.googleusercontent\\.com"
    ,"Google OAuth Access Token": "ya29\\.[0-9A-Za-z\\-_]+"
    ,"Google YouTube API Key": "AIza[0-9A-Za-z\\-_]{35}"
    ,"Google YouTube OAuth": "[0-9]+-[0-9A-Za-z_]{32}\\.apps\\.googleusercontent\\.com"
    ,"Heroku API Key": "[h|H][e|E][r|R][o|O][k|K][u|U].*[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}"
    ,"MailChimp API Key": "[0-9a-f]{32}-us[0-9]{1,2}"
    ,"Mailgun API Key": "key-[0-9a-zA-Z]{32}"
    ,"Password in URL": "[a-zA-Z]{3,10}://[^/\\s:@]{3,20}:[^/\\s:@]{3,20}@.{1,100}[\"'\\s]"
    ,"PayPal Braintree Access Token": "access_token\\$production\\$[0-9a-z]{16}\\$[0-9a-f]{32}"
    ,"Picatic API Key": "sk_live_[0-9a-z]{32}"
    ,"Slack Webhook": "https://hooks.slack.com/services/T[a-zA-Z0-9_]*/B[a-zA-Z0-9_]*/[a-zA-Z0-9_]*"
    ,"Stripe API Key": "sk_live_[0-9a-zA-Z]{24}"
    ,"Stripe Restricted API Key": "rk_live_[0-9a-zA-Z]{24}"
    ,"Square Access Token": "sq0atp-[0-9A-Za-z\\-_]{22}"
    ,"Square OAuth Secret": "sq0csp-[0-9A-Za-z\\-_]{43}"
    ,"Twilio API Key": "SK[0-9a-fA-F]{32}"
    ,"Twitter Access Token": "[t|T][w|W][i|I][t|T][t|T][e|E][r|R].*[1-9][0-9]+-[0-9a-zA-Z]{40}"
    ,"Twitter OAuth": '[t|T][w|W][i|I][t|T][t|T][e|E][r|R].*[''|"][0-9a-zA-Z]{35,44}[''|"]'
}

# Compile all the
for (k, v) in rules.items():
    rules[k] = re.compile(v)


def handler(f):
    def wrapper(*args, **kwargs):
        (severity, desc) = f(*args, **kwargs)
        if severity == 1:
            warnings.warn("SECRET: Possible secret found; this is not secure.\n%s" % yaml.dump(desc))
        elif severity >= 2:
            warnings.warn("SECRET: Likely secret found; this is not secure.\n%s" % yaml.dump(desc))
        return (severity, desc)
    return wrapper

@handler
def detect_secret(path, value, max_entropy=MAX_ENTROPY):
    """
    Args:
        path: the path leading to the key-value
        value: the actual value (bool, int, str)

    Returns:
        (int, dict) where int ∈ {0,1,2} for none,soft,hard secret.
        the dict contains a description, if there is one.
    """
    human_path = '.'.join(("[%d]" % v if isinstance(v, int) else v for v in path))

    def mask(s):
        """
        supersecretpassword -> "***************word"
        """
        return "".join(
            c if i < 4 else "*"
            for (i, c) in enumerate(s[::-1])
        )[::-1]


    # Only strings are secret
    if not isinstance(value, str):
        return 0, {}

    # It's already escaped
    if value.startswith('{{') and value.endswith('}}'):
        return 0, {}

    # Using an actual k8s secret (by reference)
    if 'secretKeyRef' in path:
        return 0, {}

    # Things like env-var NAMES, and image names
    # tend to trigger the entropy checker
    for ending in SAFE_ENDINGS:
        if path[-1].lower().endswith(ending):
            return 0, {}

    # Check the regexes - HARD violations
    for (k, regexp) in rules.items():
        if regexp.match(value):
            return 2, {
                "key": human_path,
                "violation": k,
            }

    # Check the entropy - SOFT violations
    h = None
    if len(value) >= 8:
        for alphabet in (HEX_CHARS, BASE64_CHARS, ASCII_CHARS):
            if set(value.upper()) <= alphabet:
                h = shannon_entropy(value, alphabet)
                if h > max_entropy:
                    return 1, {
                        "key": human_path,
                        'violation': f'Exceeded max entropy {h} > {max_entropy}',
                        'entropy': h,
                        'value': mask(value)
                    }

    # fallthrough
    return 0, {}




def check_for_secrets(workflow) -> int:
    """
    Recurse through a workflow and scan keys for secrets.
    Apply both entropy and regexp baesd checks. (Based on TruffleHog)

    Args:
        workflow: the workflow dictionary

    Returns:
        Count of the number of potential secrets
    """
    count = 0
    # Iterate over all keys
    for (path, key) in traversal(workflow):
        (severity, desc) = detect_secret(path, key)
        if severity != 0:
            count += 1
    return count



if __name__ == '__main__':
    with open('test.yaml') as f:
        check_for_secrets(yaml.load(f, Loader=yaml.BaseLoader))
