# run from within ./kubernetes_platform/python
PKG_ROOT=$(pwd)
REPO_ROOT=$(dirname $(dirname $PKG_ROOT))
echo $REPO_ROOT

echo "Generating Python protobuf code..."
pushd "$PKG_ROOT/.."
make clean-python python
popd

SETUPPY_VERSION=$(python -c 'from kfp.kubernetes.__init__ import __version__; print(__version__)')

if [ -z "$KFP_KUBERNETES_VERSION" ]
then
    echo "Set \$KFP_KUBERNETES_VERSION to use this script. Got empty variable."
elif [[ "$KFP_KUBERNETES_VERSION" != "$SETUPPY_VERSION" ]]
then
    echo "\$KFP_KUBERNETES_VERSION '$KFP_KUBERNETES_VERSION' does not match version in setup.py '$SETUPPY_VERSION'."
else
    echo "Got version $KFP_KUBERNETES_VERSION from env var \$KFP_KUBERNETES_VERSION"

    BRANCH_NAME=kfp-kubernetes-$KFP_KUBERNETES_VERSION
    echo "Creating release branch $BRANCH_NAME..."
    git checkout -b $BRANCH_NAME

    echo "Moving .readthedocs.yml to root..."
    # required for this branch because readthedocs only supports on docs build per repo
    # and the default is currently the KFP SDK
    # GCPC uses this pattern in this repo as well
    mv $PKG_ROOT/docs/.readthedocs.yml $REPO_ROOT/.readthedocs.yml
    rm $REPO_ROOT/kubernetes_platform/.gitignore


    echo "\nNext steps:\n\t- Add the version number to $PKG_ROOT/docs/conf.py\n\t- Add and commit the changes in this branch using 'git add $REPO_ROOT && git commit -m 'update for release' --no-verify'\n\t- Push branch using 'git push --set-upstream upstream $BRANCH_NAME'"
fi
