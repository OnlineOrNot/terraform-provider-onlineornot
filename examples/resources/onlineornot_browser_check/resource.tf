# Scripted mode: upload one self-contained `@playwright/test` file.
resource "onlineornot_browser_check" "homepage" {
  name          = "Homepage Playwright check"
  script        = file("${path.module}/homepage.spec.js")
  test_interval = 300
  test_regions  = ["aws:us-east-1"]
}

# URL mode: load a page without a test file.
resource "onlineornot_browser_check" "page_load" {
  name          = "Homepage page load"
  url           = "https://example.com"
  test_interval = 300
  test_regions  = ["aws:us-east-1"]
}
