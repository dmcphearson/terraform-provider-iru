package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// idIdentityModel is the resource identity for resources keyed by a single Iru ID.
type idIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func idIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Unique identifier of the object in Iru.",
			},
		},
	}
}

// setIDIdentity writes the identity after every Create, Read and Update so it is
// populated even when state came from an ID-based import or predates identity
// support. Identity is never read back from the request: it may be null.
func setIDIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, id types.String) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, idIdentityModel{ID: id})
}
