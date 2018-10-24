data "terraform_remote_state" "vpc" {
  backend = "s3"

  config {
    bucket = "${var.bucket}"
    key    = "terraform/network/vpc.tfstate"
    region = "${var.region}"
  }
}

data "terraform_remote_state" "subnets" {
  backend = "s3"

  config {
    bucket = "${var.bucket}"
    key    = "terraform/network/subnets.tfstate"
    region = "${var.region}"
  }
}

terraform {
  backend "s3" {
    key = "terraform/jenkins/content-server.tfstate"
  }
}
