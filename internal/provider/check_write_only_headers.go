package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateWriteOnlyHeaders(data *checkModel, diags *diag.Diagnostics) {
	headers, version := data.WriteOnlyHeaders, data.WriteOnlyHeadersVersion
	if !headers.IsUnknown() && !version.IsUnknown() && headers.IsNull() != version.IsNull() {
		diags.AddAttributeError(path.Root("write_only_headers"), "Write-only headers require a version", "Configure write_only_headers and write_only_headers_version together, or omit both.")
	}
	if !data.Headers.IsNull() && (!headers.IsNull() || !version.IsNull()) {
		diags.AddAttributeError(path.Root("write_only_headers"), "Conflicting header modes", "Configure either headers or write_only_headers, never both.")
	}
	if !version.IsNull() && !version.IsUnknown() && version.ValueInt64() < 1 {
		diags.AddAttributeError(path.Root("write_only_headers_version"), "Invalid header version", "Use a positive integer and change it whenever secret headers must be sent.")
	}
}

// Extract only from configuration at apply time, never from a plan or state.
// Reject null/unknown elements without diagnostics that could echo values.
func writeOnlyHeaders(ctx context.Context, headers types.Map, diags *diag.Diagnostics) map[string]string {
	if headers.IsNull() || headers.IsUnknown() {
		diags.AddError("Unavailable write-only headers", "Write-only headers must be known during apply.")
		return nil
	}
	for _, value := range headers.Elements() {
		if value.IsNull() || value.IsUnknown() {
			diags.AddError("Invalid write-only headers", "Every header value must be a known, non-null string during apply.")
			return nil
		}
	}
	result := map[string]string{}
	diags.Append(headers.ElementsAs(ctx, &result, false)...)
	return result
}
