#-- terraform/nagios/variables.tf
variable "env" {
  description = "Environment name (e.g. Ops, Dev, Int, Stage, Preview, Prod)"
}

variable "region" {
  description = "AWS region name (e.g. us-east-1, us-west-2, etc)"
}

variable "bucket" {
  description = "state bucket"
}

variable "execution_role_arn" {
  description = "The Amazon Resource Name (ARN) of the task execution role"
}

variable "security_groups" {
  description = "Security group IDs"
  default = []
}

variable "assign_public_ip" {
  description = "Security group IDs"
}

variable "alb_container_name" {
  description = "Name of the container receiving traffic from internet via alb."
}

variable "alb_container_port" {
  description = "Port number running on the container receiving traffic from internet."
}

variable "family" {
  description = "Task definition family"
}

variable "cluster" {
  description = "Cluster name"
}

variable "tg_arn" {
  description = "arn of the alb previously created"
}
