#-- terraform/content-service/main.tf
provider "aws" {
  region = "${var.region}"
}

resource "aws_ecs_task_definition" "this" {
  family = "${var.family}-${var.env}"
  network_mode = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu = 4096
  memory = 8192
  container_definitions = "${file("../config/${var.region}/${var.env}/content-service.json")}"
  execution_role_arn = "${var.execution_role_arn}"
}

resource "aws_ecs_service" "this" {
  name            = "${var.alb_container_name}-${var.env}"
  cluster         = "${var.cluster}-${var.env}"
  task_definition = "${aws_ecs_task_definition.this.family}:${aws_ecs_task_definition.this.revision}"
  launch_type     = "FARGATE"
  desired_count   = 1

  network_configuration {
    security_groups = "${var.security_groups}"
    subnets         = ["${data.terraform_remote_state.subnets.app_subnets_ids}"]
    assign_public_ip = "${var.assign_public_ip}"
  }

  load_balancer {
    target_group_arn = "${var.tg_arn}"
    container_name   = "${var.alb_container_name}-${var.env}"
    container_port   = "${var.alb_container_port}"
  }
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "content-service-${var.env}"
  retention_in_days = "14"
  tags {
    Name        = "content-service-${var.env}"
    Environment = "${var.env}"
    Creator     = "terraform"
  }
}
