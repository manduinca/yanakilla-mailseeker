output "public_ip" {
  description = "IP pública de la instancia"
  value       = module.app_server.public_ip
}

output "app_url" {
  description = "URL de la aplicación"
  value       = "http://${module.app_server.public_ip}"
}

output "ssh_command" {
  description = "Comando para conectarse por SSH"
  value       = "ssh ec2-user@${module.app_server.public_ip}"
}

output "instance_id" {
  description = "ID de la instancia EC2"
  value       = module.app_server.instance_id
}
