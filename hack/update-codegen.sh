set -o errexit
set -o nounset
set -o pipefail

if [ -z "${GOPATH:-}" ]; then
  export GOPATH=$(go env GOPATH)
fi

vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/pdeslaur/kube-pat/pkg/client \
    github.com/pdeslaur/kube-pat/pkg/apis \
    portaddresstranslation:v1beta1
