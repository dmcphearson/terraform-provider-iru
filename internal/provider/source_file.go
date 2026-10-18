package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Resources that upload a local file (source) record its SHA256 at plan time in
// source_sha256 and upload only when it changes.

// needsUpload reports whether a file with hash sum differs from what Iru already
// holds. After an import, or when switching from file_key to source, the prior
// source hash is null, so the API's sha256 is compared too: adopting an item whose
// binary already matches does not re-upload it.
func needsUpload(priorSourceSHA256, apiSHA256 types.String, sum string) bool {
	return sum != priorSourceSHA256.ValueString() && sum != apiSHA256.ValueString()
}

// checkSourceUnchanged fails when the file at source no longer matches the hash in
// the plan, so the binary that ships is the one the plan showed.
func checkSourceUnchanged(source, plannedSHA256 types.String, diags *diag.Diagnostics) bool {
	sum, err := fileSHA256(source.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("source"), "Cannot read source file", err.Error())
		return false
	}
	if !plannedSHA256.IsUnknown() && sum != plannedSHA256.ValueString() {
		diags.AddAttributeError(path.Root("source"), "Source file changed since plan",
			fmt.Sprintf("%s has SHA256 %s, the plan expected %s. Run plan again.",
				source.ValueString(), sum, plannedSHA256.ValueString()))
		return false
	}
	return true
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
