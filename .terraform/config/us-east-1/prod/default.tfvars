region = "us-east-1"
env = "prod"
cluster = "cl-worldteam"
execution_role_arn = "arn:aws:iam::245402993223:role/ecsTaskExecutionRole"
certificate_arn = "arn:aws:acm:us-east-1:245402993223:certificate/c5e285a2-c0e3-4a33-b93e-e4541eda4ea6"
security_groups = ["sg-079c5e0cc8c3bf6c4"]
assign_public_ip = false
alb_name = "alb-content-service"
tg_name = "tg-content-service"
health_check_path = "/"
listener_port="443"
alb_container_name="content-service"
alb_container_port="8000"
family="td-content-service"