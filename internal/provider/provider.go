package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &flokinetProvider{}

type flokinetProvider struct{ version string }

type flokinetProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	APIToken types.String `tfsdk:"api_token"`
}

type providerConfig struct {
	Host     string
	Port     int64
	Username string
	APIToken string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &flokinetProvider{version: version} }
}

func (p *flokinetProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "flokinet"
	resp.Version = p.version
}

func (p *flokinetProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"host":      schema.StringAttribute{Optional: true, Description: "cPanel host. Env: FLOKI_CPANEL_HOST"},
		"port":      schema.Int64Attribute{Optional: true, Description: "cPanel port. Env: FLOKI_CPANEL_PORT (default 2083)"},
		"username":  schema.StringAttribute{Optional: true, Description: "cPanel username. Env: FLOKI_CPANEL_USERNAME"},
		"api_token": schema.StringAttribute{Optional: true, Sensitive: true, Description: "cPanel API token. Env: FLOKI_CPANEL_API_TOKEN"},
	}}
}

func (p *flokinetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg flokinetProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("FLOKI_CPANEL_HOST")
	if !cfg.Host.IsNull() {
		host = cfg.Host.ValueString()
	}
	username := os.Getenv("FLOKI_CPANEL_USERNAME")
	if !cfg.Username.IsNull() {
		username = cfg.Username.ValueString()
	}
	token := os.Getenv("FLOKI_CPANEL_API_TOKEN")
	if !cfg.APIToken.IsNull() {
		token = cfg.APIToken.ValueString()
	}

	port := int64(2083)
	if v := os.Getenv("FLOKI_CPANEL_PORT"); v != "" {
		// fallback parse omitted; default 2083
	}
	if !cfg.Port.IsNull() {
		port = cfg.Port.ValueInt64()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(path.Root("host"), "Missing host", "Set host or FLOKI_CPANEL_HOST")
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "Missing username", "Set username or FLOKI_CPANEL_USERNAME")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_token"), "Missing api_token", "Set api_token or FLOKI_CPANEL_API_TOKEN")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	pc := &providerConfig{Host: host, Port: port, Username: username, APIToken: token}
	resp.ResourceData = pc
}

func (p *flokinetProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewAddonDomainResource}
}

func (p *flokinetProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
