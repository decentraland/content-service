#-- terraform/content-service/main.tf
provider "aws" {
  region = "${var.region}"
}

resource "aws_alb" "this" {
  name            = "${var.alb_name}-${var.env}"
  subnets         = ["${data.terraform_remote_state.subnets.app_subnets_ids}"]
  security_groups = "${var.security_groups}"
}

resource "aws_alb_target_group" "this" {
  name        = "${var.tg_name}-${var.env}"
  port        = "${var.alb_container_port}"
  protocol    = "HTTP"
  vpc_id      = "${data.terraform_remote_state.vpc.vpc_id}"
  target_type = "ip"

  health_check {
    healthy_threshold   = "3"
    interval            = "30"
    protocol            = "HTTP"
    matcher             = "200"
    timeout             = "3"
    path                = "${var.health_check_path}"
    unhealthy_threshold = "2"
  }
}

resource "aws_alb_listener" "this" {
  load_balancer_arn = "${aws_alb.this.id}"
  port              = "${var.listener_port}"
  protocol          = "HTTPS"
  certificate_arn   = "${var.certificate_arn}"

  default_action {
    target_group_arn = "${aws_alb_target_group.this.id}"
    type             = "forward"
  }
}

resource "aws_ecs_task_definition" "this" {
  family = "${var.family}-${var.env}"
  network_mode = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu = 256
  memory = 2048
  container_definitions = "${file("../config/${var.region}/${var.env}/container_definition/content-service.json")}"
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
    subnets         = ["${data.terraform_remote_state.subnets.public_subnets_ids}"]
    assign_public_ip = "${var.assign_public_ip}"
  }

  load_balancer {
    target_group_arn = "${aws_alb_target_group.this.id}"
    container_name   = "${aws_ecs_service.this.name}"
    container_port   = "${var.alb_container_port}"
  }

  depends_on = [
    "aws_alb_listener.this",
    ]

}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/fargate/service/content-service/content-service-${var.env}"
  retention_in_days = "14"
  tags {
    Name        = "content-service-${var.env}"
    Environment = "${var.env}"
    Creator     = "terraform"
  }
}
