terraform {
  required_version = ">= 1.4.0"
}

resource "terraform_data" "broken" {
  input = {
    purpose = "produce a deterministic ERROR status for tf-drift examples"
