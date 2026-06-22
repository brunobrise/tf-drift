terraform {
  required_version = ">= 1.4.0"
}

resource "terraform_data" "example" {
  input = {
    purpose = "produce a deterministic PLANNED status for tf-drift examples"
  }
}
