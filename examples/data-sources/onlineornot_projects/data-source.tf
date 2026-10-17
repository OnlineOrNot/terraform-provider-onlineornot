data "onlineornot_projects" "all" {}

output "project_ids" {
  value = { for project in data.onlineornot_projects.all.projects : project.name => project.id }
}
