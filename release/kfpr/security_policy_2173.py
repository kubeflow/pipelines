"""Exact historical 2.17.3 approvals, grouped without widening their scope.

Recorded from the merged preflight on 2026-10-22. Each group binds a
scope, artifact, and reason; every package version lists only its
approved advisories. New advisories, versions, targets, and releases
remain blocking. Images are never included in this historical policy.
"""

# (scope, artifact, reason) -> {(package, installed version): advisories}
_GROUPS = {
    ('python', '3.9', 'python39-compatibility'): {
        ('requests', '2.32.5'): ('PYSEC-2026-2275',),
        ('urllib3', '2.6.3'): (
            'PYSEC-2026-141',
            'PYSEC-2026-142',
            'PYSEC-2026-4175',
            'PYSEC-2026-4176',
            'PYSEC-2026-4177',
        ),
    },
    (
        'source',
        '.github/resources/scripts/kfp-readiness/requirements.txt',
        'nonrelease-tooling',
    ): {
        ('urllib3', '2.6.3'): (
            'CVE-2026-44431',
            'CVE-2026-44432',
            'CVE-2026-97687',
            'CVE-2026-97688',
            'CVE-2026-97689',
        ),
        ('urllib3', '2.7.0'): (
            'CVE-2026-97687',
            'CVE-2026-97688',
            'CVE-2026-97689',
        ),
    },
    (
        'source',
        'test/frontend-integration-test/package-lock.json',
        'nonrelease-tooling',
    ): {
        ('basic-ftp', '5.3.1'): ('CVE-2026-102990',),
        ('brace-expansion', '1.1.15'): (
            'CVE-2026-102276',
            'CVE-2026-102277',
            'CVE-2026-102278',
            'CVE-2026-13149',
            'CVE-2026-14257',
            'CVE-2026-69152',
        ),
        ('brace-expansion', '2.1.1'): (
            'CVE-2026-102276',
            'CVE-2026-102277',
            'CVE-2026-102278',
            'CVE-2026-13149',
            'CVE-2026-14257',
            'CVE-2026-69152',
        ),
        ('brace-expansion', '5.0.6'): (
            'CVE-2026-102276',
            'CVE-2026-102277',
            'CVE-2026-102278',
            'CVE-2026-13149',
            'CVE-2026-14257',
            'CVE-2026-69152',
        ),
        ('deepmerge-ts', '7.1.5'): ('CVE-2026-40345',),
        ('ip-address', '10.2.0'): (
            'CVE-2026-101910',
            'CVE-2026-101911',
            'CVE-2026-101912',
            'CVE-2026-101913',
            'CVE-2026-54272',
            'CVE-2026-69192',
            'CVE-2026-69198',
        ),
        ('js-yaml', '4.2.0'): (
            'CVE-2026-59869',
            'CVE-2026-84375',
            'GHSA-5p4m-2wfm-xmqj',
        ),
        ('undici', '7.28.0'): (
            'CVE-2026-13697',
            'CVE-2026-14643',
            'CVE-2026-15157',
            'CVE-2026-16728',
            'CVE-2026-16729',
            'CVE-2026-18149',
            'CVE-2026-18540',
            'CVE-2026-19534',
            'CVE-2026-84890',
            'CVE-2026-84933',
            'CVE-2026-84947',
            'CVE-2026-84961',
            'CVE-2026-85008',
            'CVE-2026-85014',
            'CVE-2026-85024',
        ),
        ('undici', '6.27.0'): (
            'CVE-2026-15157',
            'CVE-2026-16728',
            'CVE-2026-16729',
            'CVE-2026-18540',
            'CVE-2026-19534',
            'CVE-2026-85024',
        ),
    },
    ('source', 'test/tools/project-cleaner/go.mod', 'nonrelease-tooling'): {
        ('golang.org/x/crypto', 'v0.52.0'): (
            'CVE-2026-56854',
            'CVE-2026-56855',
            'CVE-2026-78662',
        ),
        ('golang.org/x/net', 'v0.55.0'): (
            'CVE-2026-46600',
            'CVE-2026-78659',
            'CVE-2026-78660',
            'CVE-2026-78663',
            'CVE-2026-78669',
            'CVE-2026-97032',
        ),
        ('golang.org/x/text', 'v0.37.0'): ('CVE-2026-56851', 'CVE-2026-56852'),
        ('google.golang.org/grpc', 'v1.79.3'): (
            'CVE-2026-84303',
            'CVE-2026-84304',
            'CVE-2026-84445',
            'GHSA-hrxh-6v49-42gf',
        ),
    },
}

APPROVED_2173 = tuple(
    (scope, artifact, package, advisory, version, reason)
    for (scope, artifact, reason), packages in _GROUPS.items()
    for (package, version), advisories in packages.items()
    for advisory in advisories
)
