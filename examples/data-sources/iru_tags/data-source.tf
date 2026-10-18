data "iru_tags" "engineering" {
  name = "Engineering"
}

# Reference the matching ID:
#   one(data.iru_tags.engineering.results).id
