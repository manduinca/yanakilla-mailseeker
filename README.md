# Yanakilla

Buscador de correos electrónicos sobre ZincSearch. Indexa una base de correos en formato RFC 5322 y expone una interfaz web para consultarla.

## Arquitectura

```
cmd/indexer      recorre el directorio, parsea y carga a ZincSearch
cmd/yanakilla    servidor HTTP: API REST + frontend embebido
internal/mailparse   parseo de correos con net/mail
internal/zinc        cliente de ZincSearch (bulk e índices)
internal/api         handlers y router chi
web                  aplicación Vue 3 + Tailwind
```

El frontend compilado se embebe en el binario con `go:embed`, así que el servidor se distribuye como un único ejecutable.

## Stack

| Capa | Tecnología |
|------|------------|
| Backend | Go |
| Motor de búsqueda | ZincSearch |
| Router HTTP | chi |
| Frontend | Vue 3 |
| CSS | Tailwind |

La única dependencia externa del backend es `chi`. El middleware de CORS está implementado en el proyecto para no añadir librerías.

## Requisitos

- Go 1.22 o superior
- Node 20 o superior
- Docker

## Puesta en marcha

Levantar ZincSearch:

```bash
docker compose up -d
```

Compilar el frontend y los binarios:

```bash
cd web && npm install && npm run build && cd ..
go build -o bin/indexer ./cmd/indexer
go build -o bin/yanakilla ./cmd/yanakilla
```

Indexar una base de correos. El indexador acepta tanto el árbol de directorios original como el export en CSV, y decide según lo que reciba:

```bash
./bin/indexer enron_mail_20110402
./bin/indexer emails.csv
```

El CSV debe tener las columnas `file` y `message`, donde `file` es la ruta original del correo y `message` su contenido completo. Para reconstruir el árbol de directorios a partir del CSV:

```bash
./bin/indexer -extract enron_mail_20110402 emails.csv
```

Levantar el servidor:

```bash
./bin/yanakilla -port 3000
```

```
Yanakilla is running in http://localhost:3000
```

## Opciones del indexador

| Flag | Por defecto | Descripción |
|------|-------------|-------------|
| `-workers` | núcleos disponibles | goroutines de parseo e ingesta |
| `-batch` | 1000 | documentos por request bulk |
| `-index` | emails | nombre del índice |
| `-zinc` | http://localhost:4080 | URL de ZincSearch |
| `-cpuprofile` | | ruta donde escribir el perfil de CPU |
| `-memprofile` | | ruta donde escribir el perfil de memoria |

## API

`GET /api/search`

| Parámetro | Descripción |
|-----------|-------------|
| `q` | término a buscar; vacío devuelve todos |
| `field` | limita la búsqueda a un campo (`subject`, `from`, `to`, `content`) |
| `from` | desplazamiento para paginación |
| `size` | resultados por página, máximo 100 |

```bash
curl "http://localhost:3000/api/search?q=manipulated&size=20"
```

`GET /api/health` devuelve el estado del servicio y el índice activo.

## Profiling

El indexador escribe perfiles de CPU y memoria cuando se le pasan las rutas correspondientes:

```bash
./bin/indexer -cpuprofile profiles/cpu.prof -memprofile profiles/mem.prof enron_mail_20110402
go tool pprof -png -output profiles/cpu.png profiles/cpu.prof
go tool pprof -top profiles/cpu.prof
```

### Hallazgos

Medición sobre 100 000 correos de la base de Enron (228 MB de contenido) en Apple Silicon.

| Configuración | Duración | Throughput |
|---------------|----------|------------|
| 1 worker, batch 100 | 15.52 s | 6 444 docs/s |
| 8 workers, batch 1000 | 9.41 s | 10 622 docs/s |

El perfil no solo mide la mejora, también muestra cómo se mueve el cuello de botella. En el baseline la CPU está casi todo el tiempo esperando:

```
Duration: 15.67s, Total samples = 4.36s (27.82%)

      flat  flat%   sum%        cum   cum%
     790ms 18.12% 18.12%      790ms  runtime.pthread_cond_signal
     730ms 16.74% 34.86%      730ms  syscall.rawsyscalln
     710ms 16.28% 51.15%      710ms  runtime.madvise
     500ms 11.47% 62.61%      500ms  runtime.usleep
```

Con la CPU activa solo el 28 % de la duración y el grueso de las muestras en sincronización y syscalls, el proceso está limitado por entrada/salida, no por cómputo: espera respuestas HTTP de ZincSearch.

Al pasar a 8 workers el perfil cambia de forma. Desaparece la espera y emerge el trabajo real de preparar cada lote:

```
Duration: 9.56s, Total samples = 3.34s (34.95%)

      flat  flat%   sum%        cum   cum%
     590ms 17.66% 17.66%      590ms  syscall.rawsyscalln
     540ms 16.17% 33.83%      540ms  runtime.memclrNoHeapPointers
     500ms 14.97% 48.80%      500ms  runtime.memmove
     420ms 12.57% 61.38%      480ms  encoding/json.appendString
```

El cuello de botella se desplazó de *esperar la red* a *serializar JSON y mover memoria*. Ese es el límite práctico de esta arquitectura: a partir de aquí, ganar más velocidad exigiría reducir el coste de serialización, no añadir más concurrencia.

De ahí las dos decisiones de diseño del indexador:

1. **Cada worker mantiene su propio lote y lo envía él mismo.** Si un único recolector centralizara los envíos, las peticiones HTTP se serializarían y volverían a ser el cuello de botella.
2. **Reutilización de conexiones.** El cliente configura `MaxIdleConnsPerHost` para evitar renegociar TCP en cada request del bulk.

El tamaño del lote domina sobre el número de workers: pasar de 100 a 1000 documentos por request reduce el número de viajes de red en un orden de magnitud.

### Formato de los correos

La base completa se indexa en unos 53 s (517 374 correos, 910 MB). De ellos, 27 correos no cumplen el RFC 5322: tienen asuntos de varias líneas cuya continuación no está indentada, por lo que `net/mail` interpreta la segunda línea como una cabecera nueva y falla.

```
Subject: Call Laddie for house party:
1. Mom &dad
```

En lugar de descartarlos, el parser reintenta una vez: si el primer intento falla, re-indenta las líneas que no son inicio de cabecera válido y vuelve a parsear. Los correos bien formados nunca pasan por ese camino.

## Desarrollo del frontend

```bash
cd web && npm run dev
```

Vite levanta en el puerto 5173 y hace proxy de `/api` hacia el servidor Go en el 3000.
