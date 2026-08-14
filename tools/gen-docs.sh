#!/usr/bin/env bash
# Regenerate docs/ with tfplugindocs, using tofu instead of terraform.
#
# tfplugindocs shells out to a `terraform` binary and downloads one if PATH has
# none. tofu can't stand in directly: tfplugindocs stages the provider under
# registry.terraform.io, which tofu won't resolve. So take the other route it
# offers, --providers-schema, and produce that JSON with tofu ourselves via
# dev_overrides. Same output, no terraform.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin" "$work/cfg"

# dev_overrides keys on the address, and tofu finds the plugin by filename, so
# the binary has to be named for the address we declare below.
go build -o "$work/bin/terraform-provider-artifactkeeper" "$root"

cat > "$work/tofurc" <<EOF
provider_installation {
  dev_overrides { "hashicorp/artifactkeeper" = "$work/bin" }
  direct {}
}
EOF

cat > "$work/cfg/main.tf" <<'EOF'
terraform {
  required_providers {
    artifactkeeper = {
      source = "hashicorp/artifactkeeper"
    }
  }
}
EOF

(cd "$work/cfg" && TF_CLI_CONFIG_FILE="$work/tofurc" tofu providers schema -json) > "$work/schema.raw.json"

# tofu keys the schema under registry.opentofu.org; tfplugindocs looks it up
# under registry.terraform.io. Only the key differs.
python3 - "$work/schema.raw.json" "$work/schema.json" <<'EOF'
import json, sys
d = json.load(open(sys.argv[1]))
s = d["provider_schemas"]
s["registry.terraform.io/hashicorp/artifactkeeper"] = s.pop(next(iter(s)))
json.dump(d, open(sys.argv[2], "w"))
EOF

cd "$root"
exec go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name artifactkeeper --providers-schema "$work/schema.json"
