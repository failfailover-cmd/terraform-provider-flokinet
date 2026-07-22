package provider

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPClientUsesConfiguredTLSServerName(t *testing.T) {
	resource := &addonDomainResource{cfg: &providerConfig{
		RequestTimeout: time.Second,
		TLSServerName:  "ro9.flokinet.is",
	}}

	client := resource.httpClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.ServerName != "ro9.flokinet.is" {
		t.Fatalf("TLS server name = %#v, want ro9.flokinet.is", transport.TLSClientConfig)
	}
	if client.Timeout != time.Second {
		t.Fatalf("timeout = %s, want %s", client.Timeout, time.Second)
	}
}

func TestHTTPClientKeepsDefaultTLSConfigWithoutServerName(t *testing.T) {
	resource := &addonDomainResource{cfg: &providerConfig{RequestTimeout: time.Second}}
	client := resource.httpClient()
	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.ServerName != "" {
		t.Fatalf("TLS server name = %q, want empty", transport.TLSClientConfig.ServerName)
	}
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = true, want false")
	}
}

func TestDeleteSubdomainPrefersLiveFullSubdomain(t *testing.T) {
	meta := &addonDomainMeta{FullSubdomain: "birdiecupeatery-com-au.kokobetdeposit.com"}
	if got := deleteSubdomain("birdiecupeatery-com-au", meta); got != meta.FullSubdomain {
		t.Fatalf("deleteSubdomain() = %q, want %q", got, meta.FullSubdomain)
	}
}

func TestDeleteSubdomainFallsBackToStateValue(t *testing.T) {
	if got := deleteSubdomain("intechs-com-au", &addonDomainMeta{}); got != "intechs-com-au" {
		t.Fatalf("deleteSubdomain() = %q, want state value", got)
	}
}

func TestAddonDomainCacheTracksCreateAndDelete(t *testing.T) {
	resource := &addonDomainResource{cfg: &providerConfig{
		addonDomainsLoaded: true,
		addonDomains:       map[string]addonDomainMeta{"existing.example": {Domain: "existing.example"}},
	}}

	resource.rememberAddonDomain("New.Example")
	if _, ok := resource.cfg.addonDomains["new.example"]; !ok {
		t.Fatal("created domain was not added to cache")
	}

	resource.forgetAddonDomain("EXISTING.example")
	if _, ok := resource.cfg.addonDomains["existing.example"]; ok {
		t.Fatal("deleted domain remained in cache")
	}
}

func TestParseAddonDomainListPreservesLiveSubdomain(t *testing.T) {
	addonDomains, err := parseAddonDomainList(`{
  "cpanelresult": {
    "data": [{
      "domain": "Example.COM",
      "fullsubdomain": "example-com.account.example"
    }]
  }
}`)
	if err != nil {
		t.Fatalf("parseAddonDomainList() error = %v", err)
	}
	meta, ok := addonDomains["example.com"]
	if !ok {
		t.Fatal("parsed domain is missing")
	}
	if meta.FullSubdomain != "example-com.account.example" {
		t.Fatalf("FullSubdomain = %q", meta.FullSubdomain)
	}
}
