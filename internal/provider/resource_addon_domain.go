package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
		"subdomain": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"docroot":   schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
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
	plan = r.normalizeAddonModel(plan)
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
	st = r.normalizeAddonModel(st)
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

func (r *addonDomainResource) normalizeAddonModel(m addonDomainModel) addonDomainModel {
	domain := strings.TrimSpace(m.Domain.ValueString())
	if domain == "" {
		return m
	}

	if m.Sub.IsNull() || m.Sub.IsUnknown() || strings.TrimSpace(m.Sub.ValueString()) == "" {
		m.Sub = types.StringValue(strings.ReplaceAll(domain, ".", "-"))
	}
	if m.Docroot.IsNull() || m.Docroot.IsUnknown() || strings.TrimSpace(m.Docroot.ValueString()) == "" {
		m.Docroot = types.StringValue(fmt.Sprintf("/home/%s/root/sites/%s", r.cfg.Username, domain))
	}
	return m
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
	ok, msg := parseCPanelResult(body)
	if !ok && !strings.Contains(strings.ToLower(msg), "already exists") {
		return fmt.Errorf("addaddondomain failed: %s; raw=%s", msg, body)
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
	_, body, err := r.call(ctx, "POST", "/json-api/cpanel", q)
	if err != nil {
		return err
	}
	ok, msg := parseCPanelResult(body)
	if !ok {
		// cPanel может вернуть warning, если домен уже отсутствует
		m := strings.ToLower(msg)
		if strings.Contains(m, "does not exist") || strings.Contains(m, "not found") {
			return nil
		}
		return fmt.Errorf("deladdondomain failed: %s; raw=%s", msg, body)
	}
	return nil
}

func (r *addonDomainResource) domainExists(ctx context.Context, domain string) (bool, error) {
	_, body, err := r.call(ctx, "GET", "/execute/DomainInfo/list_domains", nil)
	if err != nil {
		return false, err
	}
	var out struct {
		Data struct {
			AddonDomains []string `json:"addon_domains"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return strings.Contains(strings.ToLower(body), strings.ToLower(domain)), nil
	}
	for _, s := range out.Data.AddonDomains {
		if strings.EqualFold(s, domain) {
			return true, nil
		}
	}
	return false, nil
}

func (r *addonDomainResource) call(ctx context.Context, method, p string, q url.Values) (int, string, error) {
	u := fmt.Sprintf("https://%s:%d%s", r.cfg.Host, r.cfg.Port, p)
	encodedBody := ""
	if method == "GET" && len(q) > 0 {
		u += "?" + q.Encode()
	}
	if method == "POST" {
		encodedBody = q.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		var body io.Reader
		if method == "POST" {
			body = strings.NewReader(encodedBody)
		}
		req, _ := http.NewRequestWithContext(ctx, method, u, body)
		req.Header.Set("Authorization", fmt.Sprintf("cpanel %s:%s", r.cfg.Username, r.cfg.APIToken))
		if method == "POST" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		cli := &http.Client{Timeout: r.cfg.RequestTimeout}
		res, err := cli.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryableNetErr(err) || attempt == r.cfg.MaxRetries {
				return 0, "", err
			}
			time.Sleep(r.backoff(attempt, ""))
			continue
		}

		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		bodyStr := string(raw)

		if retryableStatus(res.StatusCode) && attempt < r.cfg.MaxRetries {
			time.Sleep(r.backoff(attempt, res.Header.Get("Retry-After")))
			continue
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return res.StatusCode, bodyStr, fmt.Errorf("status=%d body=%s", res.StatusCode, bodyStr)
		}
		return res.StatusCode, bodyStr, nil
	}
	return 0, "", fmt.Errorf("request retries exhausted: %w", lastErr)
}

func parseCPanelResult(raw string) (bool, string) {
	var payload struct {
		CPanelResult struct {
			Data []struct {
				Result int    `json:"result"`
				Reason string `json:"reason"`
			} `json:"data"`
			Error string `json:"error"`
		} `json:"cpanelresult"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		if strings.Contains(raw, "\"result\":1") {
			return true, ""
		}
		return false, "unparseable cPanel response"
	}
	if payload.CPanelResult.Error != "" {
		return false, payload.CPanelResult.Error
	}
	if len(payload.CPanelResult.Data) == 0 {
		return false, "empty cpanelresult.data"
	}
	if payload.CPanelResult.Data[0].Result == 1 {
		return true, payload.CPanelResult.Data[0].Reason
	}
	return false, payload.CPanelResult.Data[0].Reason
}

func retryableStatus(code int) bool {
	if code == http.StatusTooManyRequests || code == 1015 {
		return true
	}
	return code >= 500 && code <= 599
}

func isRetryableNetErr(err error) bool {
	if nerr, ok := err.(net.Error); ok {
		return nerr.Timeout() || nerr.Temporary()
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe")
}

func (r *addonDomainResource) backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if sec, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && sec > 0 {
			d := time.Duration(sec) * time.Second
			if d > r.cfg.MaxBackoff {
				return r.cfg.MaxBackoff
			}
			return d
		}
	}
	d := r.cfg.BaseBackoff * (1 << attempt)
	if d > r.cfg.MaxBackoff {
		return r.cfg.MaxBackoff
	}
	return d
}
