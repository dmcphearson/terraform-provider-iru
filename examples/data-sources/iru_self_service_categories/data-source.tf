data "iru_self_service_categories" "utilities" {
  name = "Utilities"
}

# Reference the resolved category ID, e.g. on a self-service script:
#   self_service_category_id = one(data.iru_self_service_categories.utilities.results).id
