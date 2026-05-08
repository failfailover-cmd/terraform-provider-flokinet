package provider

import (
	"context"
	"os"
	"strconv"
	"time"

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
	Host           types.String `tfsdk:"host"`
	Port           types.Int64  `tfsdk:"port"`
	Username       types.String `tfsdk:"username"`
	APIToken       types.String `tfsdk:"api_token"`
	MaxRetries     types.Int64  `tfsdk:"max_retries"`
	BaseBackoffMS  types.Int64  `tfsdk:"base_backoff_ms"`
	MaxBackoffMS   types.Int64  `tfsdk:"max_backoff_ms"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout_ms"`
}

type providerConfig struct {
	Host           string
	Port           int64
	Username       string
	APIToken       string
	MaxRetries     int
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
	RequestTimeout time.Duration
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
		"host":               schema.StringAttribute{Optional: true, Description: "cPanel host. Env: FLOKI_CPANEL_HOST"},
		"port":               schema.Int64Attribute{Optional: true, Description: "cPanel port. Env: FLOKI_CPANEL_PORT (default 2083)"},
		"username":           schema.StringAttribute{Optional: true, Description: "cPanel username. Env: FLOKI_CPANEL_USERNAME"},
		"api_token":          schema.StringAttribute{Optional: true, Sensitive: true, Description: "cPanel API token. Env: FLOKI_CPANEL_API_TOKEN"},
		"max_retries":        schema.Int64Attribute{Optional: true, Description: "Retry count for 429/5xx/network errors. Default: 6"},
		"base_backoff_ms":    schema.Int64Attribute{Optional: true, Description: "Base backoff in milliseconds. Default: 1200"},
		"max_backoff_ms":     schema.Int64Attribute{Optional: true, Description: "Max backoff in milliseconds. Default: 20000"},
		"request_timeout_ms": schema.Int64Attribute{Optional: true, Description: "HTTP request timeout in milliseconds. Default: 45000"},
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

	port := int64(envInt("FLOKI_CPANEL_PORT", 2083))
	if !cfg.Port.IsNull() {
		port = cfg.Port.ValueInt64()
	}

	maxRetries := envInt("FLOKI_MAX_RETRIES", 6)
	if !cfg.MaxRetries.IsNull() {
		maxRetries = int(cfg.MaxRetries.ValueInt64())
	}
	baseBackoffMS := envInt("FLOKI_BASE_BACKOFF_MS", 1200)
	if !cfg.BaseBackoffMS.IsNull() {
		baseBackoffMS = int(cfg.BaseBackoffMS.ValueInt64())
	}
	maxBackoffMS := envInt("FLOKI_MAX_BACKOFF_MS", 20000)
	if !cfg.MaxBackoffMS.IsNull() {
		maxBackoffMS = int(cfg.MaxBackoffMS.ValueInt64())
	}
	timeoutMS := envInt("FLOKI_REQUEST_TIMEOUT_MS", 45000)
	if !cfg.RequestTimeout.IsNull() {
		timeoutMS = int(cfg.RequestTimeout.ValueInt64())
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

	pc := &providerConfig{
		Host:           host,
		Port:           port,
		Username:       username,
		APIToken:       token,
		MaxRetries:     maxRetries,
		BaseBackoff:    time.Duration(baseBackoffMS) * time.Millisecond,
		MaxBackoff:     time.Duration(maxBackoffMS) * time.Millisecond,
		RequestTimeout: time.Duration(timeoutMS) * time.Millisecond,
	}
	resp.ResourceData = pc
}

func (p *flokinetProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewAddonDomainResource}
}

func (p *flokinetProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
