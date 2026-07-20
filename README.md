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

Indexar una base de correos:

```bash
./bin/indexer enron_mail_20110402
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

Medición sobre 40 000 correos (77 MB de contenido) en Apple Silicon.

| Configuración | Duración | Throughput |
|---------------|----------|------------|
| 1 worker, batch 100 | 12.60 s | 3 175 docs/s |
| 8 workers, batch 1000 | 5.35 s | 7 479 docs/s |

El perfil del baseline muestra por qué:

```
Duration: 12.73s, Total samples = 3.27s (25.69%)

      flat  flat%   sum%        cum   cum%
     1.98s 60.55% 60.55%      1.98s 60.55%  syscall.rawsyscalln
     0.45s 13.76% 74.31%      0.45s 13.76%  runtime.pthread_cond_wait
     0.38s 11.62% 85.93%      0.38s 11.62%  runtime.pthread_cond_signal
```

La CPU está activa apenas el 26 % de la duración total y el 60 % de las muestras corresponden a syscalls. El proceso no está limitado por cómputo sino por espera de entrada/salida: lectura de miles de archivos pequeños y requests HTTP hacia ZincSearch.

De ahí las dos decisiones de diseño del indexador:

1. **Cada worker mantiene su propio lote y lo envía él mismo.** Si un único recolector centralizara los envíos, las peticiones HTTP se serializarían y volverían a ser el cuello de botella.
2. **Reutilización de conexiones.** El cliente configura `MaxIdleConnsPerHost` para evitar renegociar TCP en cada request del bulk.

El tamaño del lote domina sobre el número de workers: pasar de 100 a 1000 documentos por request reduce el número de viajes de red en un orden de magnitud.

## Desarrollo del frontend

```bash
cd web && npm run dev
```

Vite levanta en el puerto 5173 y hace proxy de `/api` hacia el servidor Go en el 3000.
