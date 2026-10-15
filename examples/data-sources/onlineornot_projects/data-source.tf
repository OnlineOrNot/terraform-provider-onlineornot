data "onlineornot_projects" "all" {}

output "default_project_id" {
  value = one([for project in data.onlineornot_projects.all.projects : project.id if project.is_default])
}
