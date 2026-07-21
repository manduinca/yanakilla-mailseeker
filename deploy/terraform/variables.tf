variable "aws_region" {
  description = "Región de AWS donde se despliega"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Prefijo de nombre para los recursos"
  type        = string
  default     = "mailseeker"
}

variable "environment" {
  description = "Ambiente lógico (prod, dev, staging)"
  type        = string
  default     = "prod"
}

variable "instance_type" {
  description = "Tipo de instancia EC2"
  type        = string
  default     = "t3.small"
}

variable "ssh_public_key_path" {
  description = "Ruta a la clave pública que autoriza el acceso SSH"
  type        = string
}

variable "allowed_ssh_cidr" {
  description = "CIDR autorizado para SSH; usar la IP propia /32"
  type        = string
}

variable "app_port" {
  description = "Puerto HTTP público de la aplicación"
  type        = number
  default     = 80
}

variable "app_repo_url" {
  description = "Repositorio git público que la instancia clona y construye"
  type        = string
  default     = "https://github.com/manduinca/yanakilla-mailseeker.git"
}

variable "zinc_user" {
  description = "Usuario administrador de ZincSearch"
  type        = string
  default     = "admin"
}

variable "zinc_password" {
  description = "Password administrador de ZincSearch"
  type        = string
  sensitive   = true
}
