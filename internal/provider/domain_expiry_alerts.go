package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const domainExpiryAlertsDescription = "Whether to email the check's alert recipients 30, 14, 7 and 1 day before the registered domain of the check's target expires, and when it expires. " +
	"The registered domain comes from the URL host, TCP hostname or DNS domain (https://api.example.com becomes example.com). " +
	"The API rejects true when the target has no registered domain, such as an IP address, localhost or a hosting provider's subdomain like myapp.herokuapp.com, and for scripted browser checks. " +
	"Changing the target of a check with alerts enabled moves the alerts to the new target's domain, or fails if the new target has none. " +
	"Defaults to false on creation; omitting it on updates preserves the current setting."

// The default applies only on creation: preserveCheckConfiguration turns it
// into a creation default and preserves the prior value on omitted updates.
func domainExpiryAlertsAttribute() schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         domainExpiryAlertsDescription,
		MarkdownDescription: domainExpiryAlertsDescription,
		Default:             booldefault.StaticBool(false),
	}
}

func knownBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

// The API always reports the setting; absence means alerts are off.
func domainExpiryAlertsValue(value *bool) types.Bool {
	return types.BoolValue(value != nil && *value)
}
