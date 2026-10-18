data "iru_custom_scripts" "hello" {
  name = "Hello"
}

# Reference the matching ID:
#   one(data.iru_custom_scripts.hello.results).id
