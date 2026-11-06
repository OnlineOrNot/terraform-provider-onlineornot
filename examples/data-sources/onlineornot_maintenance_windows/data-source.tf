# List resources from a project and resources without a project.
data "onlineornot_maintenance_windows" "selected" {
  project_ids = ["a1b2c3d4e5f6g7h8", "unassigned"]
}

# Existing single-project filters are also supported.
data "onlineornot_maintenance_windows" "one_project" {
  project_id = "a1b2c3d4e5f6g7h8"
}
