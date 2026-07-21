output "public_ip" {
  description = "IP pública de la instancia"
  value       = aws_instance.this.public_ip
}

output "instance_id" {
  description = "ID de la instancia"
  value       = aws_instance.this.id
}

output "security_group_id" {
  description = "ID del security group"
  value       = aws_security_group.this.id
}
