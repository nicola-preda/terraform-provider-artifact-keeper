package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// The API's copy of proxy_url is not comparable to what was configured: it
// redacts credentials to *** and normalises the URL (`:3128` reads back as
// `:3128/`). egressProxyToModel must therefore keep a configured value verbatim
// and only fall back to the server's when there is none, as on import. Get it
// backwards and apply fails with "provider produced inconsistent result".
func TestEgressProxyToModelKeepsConfiguredURL(t *testing.T) {
	configured := func(url string) repositoryEgressProxyResourceModel {
		return repositoryEgressProxyResourceModel{
			RepositoryKey: types.StringValue("npm-remote"),
			ProxyURL:      types.StringValue(url),
		}
	}
	ptr := func(s string) *string { return &s }

	t.Run("credentialed URL is kept from configuration", func(t *testing.T) {
		m := egressProxyToModel(configured("http://user:pass@proxy:3128"), &client.EgressProxy{
			Mode:                       "explicit",
			ProxyURL:                   ptr("http://***@proxy:3128"),
			ProxyCredentialsConfigured: true,
		})
		if got := m.ProxyURL.ValueString(); got != "http://user:pass@proxy:3128" {
			t.Errorf("proxy_url = %q, want the configured URL, not the redacted read-back", got)
		}
		if !m.ProxyCredentialsConfigured.ValueBool() {
			t.Error("proxy_credentials_configured should come from the server")
		}
	})

	t.Run("server-normalised URL does not overwrite the configured one", func(t *testing.T) {
		// The real 1.8.0 behaviour: PUT http://proxy:3128 reads back with a
		// trailing slash. Taking that would fail the apply-consistency check.
		m := egressProxyToModel(configured("http://proxy:3128"), &client.EgressProxy{
			Mode:     "explicit",
			ProxyURL: ptr("http://proxy:3128/"),
		})
		if got := m.ProxyURL.ValueString(); got != "http://proxy:3128" {
			t.Errorf("proxy_url = %q, want the configured URL unchanged", got)
		}
	})

	t.Run("import takes the server value", func(t *testing.T) {
		// Nothing configured yet, so the server's copy is all there is.
		m := egressProxyToModel(repositoryEgressProxyResourceModel{
			RepositoryKey: types.StringValue("npm-remote"),
			ProxyURL:      types.StringNull(),
		}, &client.EgressProxy{Mode: "explicit", ProxyURL: ptr("http://proxy:3128/")})
		if got := m.ProxyURL.ValueString(); got != "http://proxy:3128/" {
			t.Errorf("proxy_url = %q, want the server value on import", got)
		}
	})

	t.Run("id mirrors the repository key", func(t *testing.T) {
		m := egressProxyToModel(configured("http://proxy:3128"), &client.EgressProxy{Mode: "inherit"})
		if m.ID.ValueString() != "npm-remote" {
			t.Errorf("id = %q, want npm-remote", m.ID.ValueString())
		}
		if !m.NoProxy.IsNull() {
			t.Errorf("no_proxy = %v, want null when the server omits it", m.NoProxy)
		}
	})
}
