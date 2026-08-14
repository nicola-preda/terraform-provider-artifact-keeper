package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// Ensure the provider satisfies the framework interface.
var _ provider.Provider = (*artifactKeeperProvider)(nil)

// ValidatedUpstreamVersion is the exact Artifact Keeper release this provider has
// been verified against. Release tags mirror the upstream MAJOR.MINOR line, with
// PATCH as the provider's own fix counter; so this constant, not the tag, is the
// precise record. Bump it in the same commit that re-validates against a release.
const ValidatedUpstreamVersion = "1.7.4"

type artifactKeeperProvider struct {
	// version is set at build time and surfaced via Metadata.
	version string
}

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &artifactKeeperProvider{version: version}
	}
}

// providerModel maps the provider configuration block.
type providerModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	Token              types.String `tfsdk:"token"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func (p *artifactKeeperProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "artifactkeeper"
	resp.Version = p.version
}

func (p *artifactKeeperProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage an Artifact Keeper instance (repositories, users, peering, SSO, and more) as code. Validated against Artifact Keeper " + ValidatedUpstreamVersion + ".",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Base URL of the Artifact Keeper instance, e.g. `https://artifact-keeper.example.com`. Use an `http://` URL for a non-TLS deployment (a local or test instance); a bare host with no scheme defaults to `https`. The `/api/v1` suffix is added automatically. May also be set via the `ARTIFACT_KEEPER_ENDPOINT` environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token (Bearer) used to authenticate. Takes precedence over `username`/`password`. May also be set via `ARTIFACT_KEEPER_TOKEN`.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for password login (`POST /api/v1/auth/login`). Used only when `token` is not set. May also be set via `ARTIFACT_KEEPER_USERNAME`.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for password login. Used only when `token` is not set. May also be set via `ARTIFACT_KEEPER_PASSWORD`.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Defaults to `false`. **Insecure**: this stops the client authenticating the server and exposes the bearer token to interception. Enable only on a trusted network against an instance with self-signed certificates. May also be set via the `ARTIFACT_KEEPER_INSECURE_SKIP_VERIFY` environment variable.",
			},
		},
	}
}

func (p *artifactKeeperProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := firstNonEmpty(cfg.Endpoint.ValueString(), os.Getenv("ARTIFACT_KEEPER_ENDPOINT"))
	token := firstNonEmpty(cfg.Token.ValueString(), os.Getenv("ARTIFACT_KEEPER_TOKEN"))
	username := firstNonEmpty(cfg.Username.ValueString(), os.Getenv("ARTIFACT_KEEPER_USERNAME"))
	password := firstNonEmpty(cfg.Password.ValueString(), os.Getenv("ARTIFACT_KEEPER_PASSWORD"))

	// insecure_skip_verify falls back to ARTIFACT_KEEPER_INSECURE_SKIP_VERIFY when
	// unset in config (config wins).
	insecure := cfg.InsecureSkipVerify.ValueBool()
	if cfg.InsecureSkipVerify.IsNull() {
		if v := os.Getenv("ARTIFACT_KEEPER_INSECURE_SKIP_VERIFY"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				insecure = b
			} else {
				resp.Diagnostics.AddAttributeError(
					path.Root("insecure_skip_verify"),
					"Invalid ARTIFACT_KEEPER_INSECURE_SKIP_VERIFY",
					"Value "+v+" is not a valid boolean; use true or false.",
				)
			}
		}
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Artifact Keeper endpoint",
			"Set the provider `endpoint` attribute or the ARTIFACT_KEEPER_ENDPOINT environment variable.",
		)
	}
	if token == "" && (username == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing Artifact Keeper credentials",
			"Provide a `token`, or both `username` and `password` (directly or via ARTIFACT_KEEPER_TOKEN / ARTIFACT_KEEPER_USERNAME / ARTIFACT_KEEPER_PASSWORD).",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(client.Config{
		Endpoint:           endpoint,
		Token:              token,
		InsecureSkipVerify: insecure,
		UserAgent:          "terraform-provider-artifact-keeper/" + p.version + " (artifact-keeper/" + ValidatedUpstreamVersion + ")",
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Artifact Keeper client", err.Error())
		return
	}

	// Exchange username/password for a token when no token was supplied.
	if token == "" {
		if err := c.Login(ctx, username, password); err != nil {
			resp.Diagnostics.AddError("Authentication against Artifact Keeper failed", err.Error())
			return
		}
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *artifactKeeperProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRepositoryResource,
		NewApiTokenResource,
		NewPeerResource,
		NewMigrationSourceResource,
		NewMigrationJobResource,
		NewSsoOidcResource,
		NewSsoLdapResource,
		NewSsoSamlResource,
		NewUserResource,
		NewGroupResource,
		NewPermissionResource,
		NewWebhookResource,
		NewServiceAccountResource,
		NewSigningKeyResource,
		NewRepoTokenResource,
		NewPromotionRuleResource,
		NewSyncPolicyResource,
		NewQualityGateResource,
		NewCurationRuleResource,
		NewLifecyclePolicyResource,
		NewCiOidcProviderResource,
		NewAgeGateResource,
		NewRemoteInstanceResource,
		NewSecurityPolicyResource,
		NewRepositoryLabelResource,
		NewProjectResource,
		NewProjectMembershipResource,
		NewLicensePolicyResource,
		NewEmailSubscriptionResource,
		NewCiOidcIdentityMappingResource,
		NewServiceAccountTokenResource,
		NewUserRoleAssignmentResource,
		NewGroupMembershipResource,
		NewPeerRepositorySubscriptionResource,
		NewSystemSettingsResource,
		NewTelemetrySettingsResource,
		NewRepositorySigningConfigResource,
		NewRepositorySecurityResource,
		NewRepositoryCacheTtlResource,
		NewRepositoryNpmScopePolicyResource,
		NewRepositoryReleaseTargetResource,
		NewRepositoryRoutingRulesResource,
		NewRepositoryPypiTrackResource,
		NewRepositoryUpstreamAuthResource,
		NewPeerNetworkProfileResource,
		NewPeerInstanceLabelResource,
		NewUserApiTokenResource,
		NewFormatHandlerResource,
		NewPluginResource,
	}
}

func (p *artifactKeeperProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRepositoryDataSource,
		NewUserDataSource,
		NewGroupDataSource,
		NewProjectDataSource,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
