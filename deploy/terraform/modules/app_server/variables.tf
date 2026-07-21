variable "name_prefix" {
  description = "Prefijo para nombrar los recursos"
  type        = string
}

variable "vpc_id" {
  description = "VPC donde se crea el security group"
  type        = string
}

variable "subnet_id" {
  description = "Subnet donde se lanza la instancia"
  type        = string
}

variable "ami_id" {
  description = "AMI base de la instancia"
  type        = string
}

variable "instance_type" {
  description = "Tipo de instancia EC2"
  type        = string
}

variable "ssh_public_key" {
  description = "Contenido de la clave pública SSH"
  type        = string
}

variable "allowed_ssh_cidr" {
  description = "CIDR autorizado para SSH"
  type        = string
}

variable "app_port" {
  description = "Puerto HTTP público de la aplicación"
  type        = number
}

variable "user_data" {
  description = "Script de arranque de la instancia"
  type        = string
}
