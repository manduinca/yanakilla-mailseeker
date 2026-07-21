# Despliegue

Infraestructura como código con Terraform. Levanta una instancia EC2 que ejecuta ZincSearch y la aplicación con Docker Compose, en la VPC por defecto de la cuenta.

## Arquitectura

```
                 puerto 80
   internet  ─────────────────►  EC2 (Amazon Linux 2023)
                                   ├── yanakilla-app   (contenedor, :3000 → :80)
                                   └── yanakilla-zinc  (contenedor, :4080 interno)
```

La instancia arranca con un script que instala Docker, clona el repositorio, construye la imagen y levanta `docker-compose.prod.yml`. El puerto 80 queda abierto a internet; el 22 solo a la IP indicada.

## Estructura

```
terraform/
├── main.tf                    provider, data sources y llamada al módulo
├── variables.tf
├── outputs.tf
├── terraform.tfvars.example   copiar a terraform.tfvars y completar
├── templates/
│   └── user_data.sh.tftpl     arranque de la instancia
└── modules/
    └── app_server/            security group, key pair e instancia
```

Los valores concretos (IP autorizada, password, clave pública) van en `terraform.tfvars`, que está en `.gitignore`. El repositorio solo contiene el ejemplo con placeholders.

## Uso

Generar el par de claves SSH:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/mailseeker-prod-key -N ""
```

Preparar las variables:

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars
# editar terraform.tfvars: allowed_ssh_cidr con la IP propia, zinc_password, etc.
```

Desplegar:

```bash
terraform init
terraform plan
terraform apply
```

Al terminar, `terraform output` muestra la URL pública. La primera vez la instancia tarda unos minutos en construir la imagen; el progreso se sigue por SSH en `/var/log/user-data.log`.

## Cargar datos

La instancia arranca con el índice vacío. Para indexar desde la propia instancia:

```bash
ssh -i ~/.ssh/mailseeker-prod-key ec2-user@<ip>
# subir un csv o directorio de correos y ejecutarlo dentro del contenedor
docker exec -i yanakilla-app indexer -zinc http://zincsearch:4080 /ruta/al/origen
```

## Destruir

```bash
terraform destroy
```

Elimina la instancia, el security group y el key pair. El costo se detiene ahí.
