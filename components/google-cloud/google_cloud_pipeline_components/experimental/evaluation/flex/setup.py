"""Setup for DataFlow Worker environments."""
import setuptools
setuptools.setup(
    name='mes-package',
    version='1.0.0',
    install_requires=[
        'tensorflow_model_analysis==0.25.0',
    ],
    packages=setuptools.find_packages(),
)
