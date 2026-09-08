# IDs below must include every current member of this scope.
# Prefer references to managed component/group IDs to establish dependencies.
resource "onlineornot_status_page_component_order" "layout" {
  status_page_id = "pageAAAA"
  component_ids  = ["componentA", "componentB"]
}
