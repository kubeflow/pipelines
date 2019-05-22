from setuptools import setup


NAME = 'kfp-arena'
VERSION = '0.6'

REQUIRES = ['kfp >= 0.1']

setup(
    name=NAME,
    version=VERSION,
    description='KubeFlow Pipelines Extended Arena SDK',
    author='cheyang',
    author_email="cheyang@163.com",
    install_requires=REQUIRES,
    packages=[
      'arena',
    ],
    classifiers=[
      'Intended Audience :: Developers',
      'Intended Audience :: Education',
      'Intended Audience :: Science/Research',
      'License :: OSI Approved :: Apache Software License',
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
    python_requires='>=3.5.3',
    include_package_data=True
)
