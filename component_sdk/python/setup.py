from setuptools import setup

PACKAGE_NAME = "kfp-component"
VERSION = '0.1'
REQUIRES = []
with open('requirements.txt') as f:
    REQUIRES = f.readlines()

with open('test-requirements.txt') as f:
    TESTS_REQUIRES = f.readlines()

setup(
    name=PACKAGE_NAME,
    version=VERSION,
    description='KubeFlow Pipelines Component SDK',
    author='google',
    install_requires=REQUIRES,
    tests_require=TESTS_REQUIRES,
    packages=[
      'kfp_component',
    ],
    classifiers=[
      'Intended Audience :: Developers',
      'Intended Audience :: Education',
      'Intended Audience :: Science/Research',
      'License :: OSI Approved :: Apache Software License',
      'Programming Language :: Python',
      'Programming Language :: Python :: 2',
      'Programming Language :: Python :: 2.7',
      'Programming Language :: Python :: 3',
      'Programming Language :: Python :: 3.5',
      'Programming Language :: Python :: 3.6',
      'Programming Language :: Python :: 3.7',
      'Topic :: Scientific/Engineering',
      'Topic :: Scientific/Engineering :: Artificial Intelligence',
      'Topic :: Software Development',
      'Topic :: Software Development :: Libraries',
      'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    include_package_data=True,
)
