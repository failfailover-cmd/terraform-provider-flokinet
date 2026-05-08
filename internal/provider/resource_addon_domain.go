package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &addonDomainResource{}
var _ resource.ResourceWithImportState = &addonDomainResource{}

type addonDomainResource struct{ cfg *providerConfig }

type addonDomainModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Sub     types.String `tfsdk:"subdomain"`
	Docroot types.String `tfsdk:"docroot"`
}

func NewAddonDomainResource() resource.Resource { return &addonDomainResource{} }

func (r *addonDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_addon_domain"
}

func (r *addonDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":        schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"domain":    schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"subdomain": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"docroot":   schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
	}}
}

func (r *addonDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*providerConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected config type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.cfg = cfg
}

func (r *addonDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan addonDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.addAddonDomain(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Floki API error", err.Error())
		return
	}
	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addonDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var st addonDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &st)...)
	if resp.Diagnostics.HasError() {
		return
	}
	exists, err := r.domainExists(ctx, st.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Floki API error", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &st)...)
}

func (r *addonDomainResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *addonDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var st addonDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &st)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.delAddonDomain(ctx, st.Domain.ValueString(), st.Sub.ValueString()); err != nil {
		resp.Diagnostics.AddError("Floki API error", err.Error())
	}
}

func (r *addonDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *addonDomainResource) addAddonDomain(ctx context.Context, d addonDomainModel) error {
	q := url.Values{}
	q.Set("cpanel_jsonapi_user", r.cfg.Username)
	q.Set("cpanel_jsonapi_apiversion", "2")
	q.Set("cpanel_jsonapi_module", "AddonDomain")
	q.Set("cpanel_jsonapi_func", "addaddondomain")
	q.Set("newdomain", d.Domain.ValueString())
	q.Set("subdomain", d.Sub.ValueString())
	q.Set("dir", d.Docroot.ValueString())
	_, body, err := r.call(ctx, "POST", "/json-api/cpanel", q)
	if err != nil {
		return err
	}
	if !strings.Contains(body, "\"result\":1") && !strings.Contains(strings.ToLower(body), "already exists") {
		return fmt.Errorf("unexpected response: %s", body)
	}
	return nil
}

func (r *addonDomainResource) delAddonDomain(ctx context.Context, domain, subdomain string) error {
	q := url.Values{}
	q.Set("cpanel_jsonapi_user", r.cfg.Username)
	q.Set("cpanel_jsonapi_apiversion", "2")
	q.Set("cpanel_jsonapi_module", "AddonDomain")
	q.Set("cpanel_jsonapi_func", "deladdondomain")
	q.Set("domain", domain)
	q.Set("subdomain", subdomain)
	_, _, err := r.call(ctx, "POST", "/json-api/cpanel", q)
	return err
}

func (r *addonDomainResource) domainExists(ctx context.Context, domain string) (bool, error) {
	_, body, err := r.call(ctx, "GET", "/execute/DomainInfo/list_domains", nil)
	if err != nil {
		return false, err
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return strings.Contains(body, domain), nil
	}
	rawData, ok := out["data"].(map[string]any)
	if !ok {
		return strings.Contains(body, domain), nil
	}
	arr, ok := rawData["addon_domains"].([]any)
	if !ok {
		return strings.Contains(body, domain), nil
	}
	for _, x := range arr {
		if s, ok := x.(string); ok && strings.EqualFold(s, domain) {
			return true, nil
		}
	}
	return false, nil
}

func (r *addonDomainResource) call(ctx context.Context, method, p string, q url.Values) (int, string, error) {
	u := fmt.Sprintf("https://%s:%d%s", r.cfg.Host, r.cfg.Port, p)
	var body io.Reader
	if method == "GET" && len(q) > 0 {
		u += "?" + q.Encode()
	}
	if method == "POST" {
		body = strings.NewReader(q.Encode())
	}
	req, _ := http.NewRequestWithContext(ctx, method, u, body)
	req.Header.Set("Authorization", fmt.Sprintf("cpanel %s:%s", r.cfg.Username, r.cfg.APIToken))
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	cli := &http.Client{Timeout: 45 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return res.StatusCode, string(raw), fmt.Errorf("status=%d body=%s", res.StatusCode, string(raw))
	}
	return res.StatusCode, string(raw), nil
}
