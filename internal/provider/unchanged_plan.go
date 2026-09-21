package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Retain computed values when they are the only differences from prior state.
func useStateForUnchangedPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, paths ...path.Path) {
	if resp.Diagnostics.HasError() || req.State.Raw.IsNull() || resp.Plan.Raw.IsNull() {
		return
	}
	plan := resp.Plan
	for _, p := range paths {
		var planned, configured, prior attr.Value
		resp.Diagnostics.Append(plan.GetAttribute(ctx, p, &planned)...)
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &configured)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !planned.IsUnknown() || !configured.IsNull() {
			continue
		}
		resp.Diagnostics.Append(plan.SetAttribute(ctx, p, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if plan.Raw.Equal(req.State.Raw) {
		resp.Plan = plan
	}
}
