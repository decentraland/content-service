region = "us-east-1"
env = "dev"
cluster = "cl-worldteam"
execution_role_arn = "arn:aws:iam::872049612737:role/ecsTaskExecutionRole"
certificate_arn = "arn:aws:acm:us-east-1:872049612737:certificate/ab0afbe5-9f13-4edc-80ef-1cc7967339eb"
security_groups = ["sg-042ee236e279c74a3"]
assign_public_ip = false
alb_name = "alb-content-service"
tg_name = "tg-content-service"
health_check_path = "/"
listener_port="443"
alb_container_name="content-service"
alb_container_port="8000"
family="td-content-service"