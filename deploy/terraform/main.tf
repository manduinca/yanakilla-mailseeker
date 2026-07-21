provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

module "app_server" {
  source = "./modules/app_server"

  name_prefix      = "${var.project_name}-${var.environment}"
  vpc_id           = data.aws_vpc.default.id
  subnet_id        = sort(data.aws_subnets.default.ids)[0]
  ami_id           = data.aws_ami.al2023.id
  instance_type    = var.instance_type
  ssh_public_key   = file(var.ssh_public_key_path)
  allowed_ssh_cidr = var.allowed_ssh_cidr
  app_port         = var.app_port

  user_data = templatefile("${path.module}/templates/user_data.sh.tftpl", {
    repo_url      = var.app_repo_url
    zinc_user     = var.zinc_user
    zinc_password = var.zinc_password
    domain        = var.domain
  })
}
