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


#When you use a alb. Esle remove this aws_ecs_task_definition
variable "alb_name" {
  description = "name of the ALB"
}

variable "tg_name" {
  description = "name of the target group"
}

variable "health_check_path" {
  description = "path of the health_check"
}

variable "listener_port" {
  description = "Port where the alb listens"
}

variable "alb_container_name" {
  description = "Name of the container receiving traffic from internet via alb."
}

variable "alb_container_port" {
  description = "Port number running on the container receiving traffic from internet."
}

variable "certificate_arn" {
  description = "SSL certificate"
}

variable "family" {
  description = "Task definition family"
}

variable "cluster" {
  description = "Cluster name"
}

variable "matcher" {
  description = "Error Matcher"
}

variable "deregistration_delay" {
  description = "The amount time for Elastic Load Balancing to wait before changing the state of a deregistering target from draining to unused."
}
