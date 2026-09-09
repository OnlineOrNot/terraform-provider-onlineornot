# Each matched file creates one monitor. These files must be independent.
resource "onlineornot_browser_check" "files" {
  for_each = fileset("${path.module}/checks", "*.spec.js")

  name          = trimsuffix(each.key, ".spec.js")
  script        = file("${path.module}/checks/${each.key}")
  test_interval = 300
  test_regions  = ["aws:us-east-1"]
}
