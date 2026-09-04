package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type operationalStateChange struct {
	field string
	value bool
}

func operationalStateChanges(paused, muted, currentPaused, currentMuted types.Bool) ([]operationalStateChange, error) {
	if !paused.IsNull() && !paused.IsUnknown() && paused.ValueBool() &&
		!muted.IsNull() && !muted.IsUnknown() && muted.ValueBool() {
		return nil, fmt.Errorf("paused and muted cannot both be true")
	}

	var pauseChange, muteChange *operationalStateChange
	if !paused.IsNull() && !paused.IsUnknown() &&
		(currentPaused.IsNull() || currentPaused.IsUnknown() || !paused.Equal(currentPaused)) {
		pauseChange = &operationalStateChange{field: "paused", value: paused.ValueBool()}
	}
	if !muted.IsNull() && !muted.IsUnknown() &&
		(currentMuted.IsNull() || currentMuted.IsUnknown() || !muted.Equal(currentMuted)) {
		muteChange = &operationalStateChange{field: "muted", value: muted.ValueBool()}
	}

	// The API accepts only one operational state field per request. Disable the
	// old state before enabling the new one when moving directly between states.
	changes := make([]operationalStateChange, 0, 2)
	if paused.ValueBool() {
		if muteChange != nil {
			changes = append(changes, *muteChange)
		}
		if pauseChange != nil {
			changes = append(changes, *pauseChange)
		}
	} else {
		if pauseChange != nil {
			changes = append(changes, *pauseChange)
		}
		if muteChange != nil {
			changes = append(changes, *muteChange)
		}
	}
	return changes, nil
}

func applyOperationalState(change operationalStateChange, paused, muted **bool) {
	value := change.value
	if change.field == "paused" {
		*paused = &value
		*muted = nil
		return
	}
	*paused = nil
	*muted = &value
}

func pausedAttribute(subject string) schema.BoolAttribute {
	description := fmt.Sprintf("Whether the %s is paused. Cannot be true when muted is true.", subject)
	return schema.BoolAttribute{Optional: true, Computed: true, Description: description, MarkdownDescription: description}
}

func mutedAttribute(subject string) schema.BoolAttribute {
	description := fmt.Sprintf("Whether alerts for the %s are muted. Cannot be true when paused is true.", subject)
	return schema.BoolAttribute{Optional: true, Computed: true, Description: description, MarkdownDescription: description}
}
