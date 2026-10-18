package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The framework validates list resources when Terraform loads the schema: each
// needs a managed resource of the same name that supports identity.
func TestProviderSchemaIncludesListResources(t *testing.T) {
	srv, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s: %s", d.Severity, d.Summary, d.Detail)
	}
	for name := range resp.ResourceSchemas {
		if _, ok := resp.ListResourceSchemas[name]; !ok {
			t.Errorf("no list resource for %s", name)
		}
	}
}
