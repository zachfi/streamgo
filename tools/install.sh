#! /bin/bash
set -e

# Receive an import, which may be a module path.  If we can list the module
# path, return it.  If we cannot, strip off the last path segment and try
# again.  Repeat until we find a module path.
function module_for_import() {
  local import=$1

  local module

  module=$(go list -m -f '{{.Path}}' "$import" 2>/dev/null)
  if [ "$module" ]; then
    echo "$module"
    return
  fi

  while [ ! "$module" ]; do
    import=$(echo "$import" | sed -e 's:/[^/]*$::')
    module=$(go list -m -f '{{.Path}}' "$import" 2>/dev/null)
    if [ "$module" ]; then
      echo "$module"
      return
    fi
  done

  return 1
}

declare -a imports

# tools.go has //go:build tools; must pass -tags tools so its imports are listed
imports=($(go list -e -tags tools -f '{{join .Imports " "}}' tools.go))
for i in "${imports[@]}"; do
  module=$(module_for_import "$i")

  # Install from current module (no @version) so tools/go.mod replace directives apply
  # (e.g. golang.org/x/tools => v0.32.0 for Go 1.26). Otherwise go install pkg@version
  # builds that module in isolation and old transitive deps like x/tools@v0.24.0 fail on Go 1.26.
  if [ "$module" ]; then
    go install "$i"
  fi

done

# Drone CLI is installed via apk (edge/testing) in the Dockerfile; ensure it's available.
if ! command -v drone >/dev/null 2>&1; then
  echo "error: drone not found (expected from apk add drone-cli in Dockerfile)" >&2
  exit 1
fi
