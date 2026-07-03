resource "iru_custom_script" "hello" {
  name                = "Hello World"
  execution_frequency = "once"
  script              = file("${path.module}/hello.sh")
  active              = true
}
