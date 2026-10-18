# Active users from every directory integration.
data "iru_users" "active" {
  archived = false
}
