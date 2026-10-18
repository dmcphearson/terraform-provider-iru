resource "iru_in_house_app" "field_app" {
  name   = "Field App"
  source = "${path.module}/FieldApp.ipa"

  runs_on_iphone = true
  runs_on_ipad   = true
  runs_on_tv     = false
}
