resource "onlineornot_browser_check" "checkout" {
  name = "Checkout flow"
  url  = "https://example.com/checkout"

  test_interval = 300
  test_regions = [
    "aws:us-east-1",
  ]
}

resource "onlineornot_browser_check" "scripted_checkout" {
  name = "Scripted checkout flow"

  script = <<-EOT
    import { test, expect } from '@playwright/test';

    test('checkout page loads', async ({ page }) => {
      await page.goto('https://example.com/checkout');
      await expect(page.getByRole('heading', { name: 'Checkout' })).toBeVisible();
    });
  EOT
}
