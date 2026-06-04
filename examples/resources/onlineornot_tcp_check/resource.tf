resource "onlineornot_tcp_check" "postgres" {
  name         = "Postgres TCP"
  tcp_hostname = "db.example.com"
  tcp_port     = 5432

  test_interval = 300
  timeout       = 10000
  test_regions = [
    "aws:us-east-1",
  ]
}

resource "onlineornot_tcp_check" "smtp_banner" {
  name         = "SMTP banner"
  tcp_hostname = "mail.example.com"
  tcp_port     = 25
  tcp_data     = "EHLO example.com\r\n"
}
