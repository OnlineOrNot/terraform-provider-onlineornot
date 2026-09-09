# IDs below must include every current member of this scope.
# Prefer references to managed component/group IDs to establish dependencies.
resource "onlineornot_status_page_group_component_order" "layout" {
  status_page_id = "pageAAAA"
  group_id       = "groupAAAA"
  component_ids  = ["componentA", "componentB"]
}
