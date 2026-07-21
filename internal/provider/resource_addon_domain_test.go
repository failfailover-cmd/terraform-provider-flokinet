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
