# List all users in your organization
data "onlineornot_users" "all" {}

output "user_emails" {
  value = [for user in data.onlineornot_users.all.users : user.email]
}
