# NodosML — Movie Recommender API 🎬

API para la PC4 de Programación Concurrente que implementa un sistema de recomendación de películas distribuido en múltiples nodos ML. Incluye:

- **MongoDB** como base de datos principal.  
- **Redis** como caché de recomendaciones.  
- **Nodos ML** independientes (micro-servicios) que calculan recomendaciones.  
- **API Gateway en Go** que coordina los nodos ML y expone endpoints REST + WebSocket.  
- **JWT + roles (`admin` / `user`)** para controlar el acceso a la API.  
- **Swagger** para documentar y probar los endpoints.  

**Resumen rápido**: la API valida JWT, aplica reglas por rol y orquesta peticiones a los nodos ML. Redis cachea resultados y MongoDB almacena usuarios, películas, ratings y objetos auxiliares.

**Contenido**

- **Arquitectura**
- **Estructura de carpetas**
- **Endpoints principales**
- **Autenticación y autorización**
- **Flujo mejorado (secuencia)**
- **Cómo levantar el proyecto**
- **Notas de despliegue**

**Arquitectura (alto nivel)**

```text
            ┌─────────────────────────────────────┐
            │           API (pc4-api)             │
            │   - Autenticación JWT               │
            │   - Endpoints /auth, /movies, /me   │
            │   - Orquestador de nodos ML         │
            └───────────────┬─────────────────────┘
                            │ HTTP (cluster)
         ┌──────────────────┼──────────────────┬──────────────────┐
         │                  │                  │                  │
 ┌───────────────┐   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
 │  mlnode1       │   │  mlnode2      │  │  mlnode3      │  │  mlnode4      │
 │  :9001         │   │  :9001        │  │  :9001        │  │  :9001        │
 │  shard 1       │   │  shard 2      │  │  shard 3      │  │  shard 4      │
 └───────┬────────┘   └───────┬───────┘  └───────┬───────┘  └───────┬───────┘
         │ MongoDB            │                │                 │
         └────────────────────┴────────────────┴─────────────────┘
                     ┌─────────────────────────────────────┐
                     │        MongoDB (pc4-mongo)          │
                     │ ratings, movies, similarities, etc. │
                     └─────────────────────────────────────┘

                     ┌─────────────────────────────────────┐
                     │        Redis (pc4-redis)            │
                     │  Caché de recomendaciones / sims    │
                     └─────────────────────────────────────┘
```

**Estructura de carpetas (resumen)**

. ├── `cmd/` — entrypoints
. │   ├── `api/` — API Gateway (puerto 8080)
. │   └── `mlnode/` — Nodo ML (puerto 9001)
. ├── `docs/` — Swagger/OpenAPI
. ├── `internal/` — lógica interna (cache, cluster, db, handlers, models, repo, services)
. ├── `docker-compose.yml` — orquestación (mongo, redis, api, mlnode1..4)
. └── `.env` — configuración opcional local

Detalles por carpeta:

- `internal/config`: carga variables de entorno (MONGO_URI, MONGO_DB, REDIS_ADDR, JWT_SECRET, HTTP_PORT, ML_NODE_ADDRS).
- `internal/db`: inicializa cliente Mongo y expone `DB()` para repositorios.
- `internal/cache`: cliente Redis y helpers de caching.
- `internal/cluster`: cliente que orquesta llamadas a los `mlnode` y combina respuestas.
- `internal/handler`: controladores HTTP, JWT middleware y WebSocket handler.
- `internal/service`: lógica de negocio (auth, movie, rating, recommend).

---

## 2. Estructura de carpetas

    .
    ├── cmd/                          # Aplicaciones ejecutables (entrypoints)
    │   ├── api/                      # API HTTP principal
    │   │   ├── Dockerfile            # Imagen Docker de la API
    │   │   └── main.go               # Punto de entrada de la API
    │   └── mlnode/                   # Nodo de cómputo ML (se replica 4 veces)
    │       ├── Dockerfile            # Imagen Docker de cada nodo ML
    │       └── main.go               # Punto de entrada de cada nodo ML
    │
    ├── docs/                         # Documentación Swagger (OpenAPI)
    │   ├── docs.go                   # Inicialización de Swagger en Go
    │   ├── swagger.json              # Esquema generado
    │   └── swagger.yaml              # Esquema editable
    │
    ├── internal/                     # Código interno (no exportable fuera del módulo)
    │   ├── cache/                    # Integración con Redis
    │   │   └── redis.go              # Cliente y helpers de caché
    │   ├── cluster/                  # Cliente para comunicarse con los nodos ML
    │   │   ├── client.go             # Lógica de orquestación y llamadas HTTP
    │   │   └── messages.go           # DTOs de petición/respuesta con nodos ML
    │   ├── config/                   # Configuración de la app
    │   │   └── config.go             # Carga de variables de entorno
    │   ├── db/                       # Integración con MongoDB
    │   │   └── mongo.go              # Cliente global de MongoDB
    │   ├── handler/                  # Capa HTTP (controladores)
    │   │   ├── auth_handler.go       # /auth/*
    │   │   ├── health_handler.go     # /health
    │   │   ├── jwt_middleware.go     # Middlewares JWT + roles
    │   │   ├── movie_handler.go      # /movies/*
    │   │   ├── rating_handler.go     # /me/ratings, /users/{id}/ratings
    │   │   └── recommend_handler.go  # /me/recommendations, /users/{id}/recommendations
    │   ├── models/                   # Modelos de dominio (Movie, User, Rating, etc.)
    │   │   ├── movie.go
    │   │   ├── rating.go
    │   │   ├── recommendation.go
    │   │   ├── similarity.go
    │   │   └── user.go
    │   ├── repository/               # Capa de acceso a datos (MongoDB)
    │   │   ├── movie_repo.go
    │   │   ├── rating_repo.go
    │   │   ├── recommendation_repo.go
    │   │   ├── similarity_repo.go
    │   │   └── user_repo.go
    │   └── service/                  # Lógica de negocio / casos de uso
    │       ├── auth_service.go
    │       ├── movie_service.go
    │       ├── rating_service.go
    │       └── recommend_service.go
    │
    ├── docker-compose.yml            # Orquestación de Mongo, Redis, API, nodos ML
    ├── go.mod                        # Definición del módulo Go
    ├── go.sum                        # Checksums de dependencias
    ├── .env                          # Config local (opcional)
    └── README.md                     # Este archivo

### 2.1. Explicación de carpetas / archivos principales

#### `cmd/api`

- `main.go`  
  Punto de entrada de la API. Hace:

  - Carga configuración (`config.Load()`).
  - Inicializa Mongo y Redis.
  - Crea los repositorios y servicios.
  - Lee direcciones de nodos ML desde `ML_NODE_ADDRS`.
  - Crea handlers HTTP.
  - Define rutas públicas y protegidas (incluyendo reglas por rol).
  - Expone Swagger en `/swagger/*`.

- `Dockerfile`  
  Construye el binario de la API en una imagen multi-stage. El stage final es una imagen Alpine ligera que ejecuta `/app/api` en el puerto `8080`.

#### `cmd/mlnode`

- `main.go`  
  Punto de entrada de cada nodo de cómputo ML. Lee variables como `NODE_ID` y `ML_NODE_ADDR`, inicia un servidor HTTP que recibe peticiones de la API, consulta Mongo y devuelve recomendaciones parciales.

- `Dockerfile`  
  Compila y empaqueta el binario `mlnode` en una imagen Alpine que expone el puerto `9001`.

#### `docs`

- `docs.go`  
  Inicializa la documentación Swagger generada por `swag init`.

- `swagger.yaml` / `swagger.json`  
  Definen el esquema OpenAPI de la API (endpoints, parámetros, modelos, etc.). Swagger UI los usa para mostrar la documentación en `/swagger/index.html`.

#### `internal/config`

- `config.go`  
  Define la estructura `Config` con:

  - `MongoURI`, `MongoDB`
  - `RedisAddr`, `RedisPass`
  - `JWTSecret`
  - `HTTPPort`

  Carga variables desde el entorno (`os.Getenv`) y, si no existen, usa valores por defecto (también deja un log de warning). Permite usar `.env` en local o variables en `docker-compose` en producción.

#### `internal/db`

- `mongo.go`  
  Inicializa el cliente global de MongoDB con el URI y DB name de `Config`.  
  Expone una función `DB()` para obtener la referencia a la base y que los repositorios puedan abrir colecciones.

#### `internal/cache`

- `redis.go`  
  Inicializa el cliente de Redis apuntando a `REDIS_ADDR`.  
  Expone helpers para guardar y leer claves (por ejemplo, recomendaciones cacheadas por usuario).  
  Es la pieza que permite que `RecommendService` no tenga que recalcular siempre todo.

#### `internal/cluster`

- `client.go`  
  Implementa un cliente HTTP para comunicarse con los nodos ML. Sabe las direcciones de cada nodo (vienen de `ML_NODE_ADDRS`) y reparte las peticiones entre ellos. También se encarga de timeouts y manejo de errores.

- `messages.go`  
  Define los DTOs/structs de las peticiones y respuestas entre la API y los nodos ML (por ejemplo, estructura de `RecItem`, payload para pedir recomendaciones, etc.). Esto asegura que ambos lados hablen el mismo “contrato”.

#### `internal/models`

Modelos del dominio central:

- `movie.go`: estructura de película (ID, título, géneros, popularidad, etc.).
- `rating.go`: estructura de rating (`userId`, `movieId`, `rating`, `timestamp`).
- `recommendation.go`: estructura para recomendaciones (`movieId`, `score`, explicación, etc.).
- `similarity.go`: estructura para guardar similitudes item-based entre películas.
- `user.go`: estructura de usuario (`userId`, `email`, `password` hasheado, `role`).

#### `internal/repository`

Capa de acceso a datos (MongoDB). Cada repo se enfoca en una colección:

- `movie_repo.go`: consultas de películas (`GetByID`, búsqueda con filtros, paginación).
- `rating_repo.go`: guardar y leer ratings (`UpsertRating`, `GetByUser`, `GetAllByUser`).
- `recommendation_repo.go`: historial de recomendaciones generadas (para auditoría o análisis).
- `similarity_repo.go`: leer / guardar similitudes item-based precomputadas.
- `user_repo.go`: operaciones sobre usuarios (crear, buscar por email, actualizar datos).

#### `internal/service`

Lógica de negocio:

- `auth_service.go`  

  - Registra usuarios nuevos (`Register`).
  - Valida credenciales (`Login`).
  - Genera tokens JWT firmados con `JWT_SECRET` incluyendo `sub`, `role` y `exp`.
  - Actualiza datos de usuario (`UpdateUser`).

- `movie_service.go`  

  - Orquesta las consultas a `movie_repo` para obtener películas por ID o buscar listados.

- `rating_service.go`  

  - Encapsula la lógica de crear/actualizar ratings.
  - Usa `rating_repo` para guardar en Mongo y para recuperar ratings de un usuario.

- `recommend_service.go`  

  - Implementa el flujo de recomendaciones:
    - Revisa caché en Redis.
    - Si no hay caché, llama al cliente de cluster para pedir resultados a los nodos ML.
    - Combina respuestas, ordena por score y las devuelve.
    - Guarda resultados en Redis y opcionalmente en `recommendation_repo`.

#### `internal/handler`

Controladores HTTP:

- `auth_handler.go`  

  - `POST /auth/register`: registro de usuarios.
  - `POST /auth/login`: login, devuelve token y datos básicos.
  - `PUT /users/{id}/update`: actualización de `email`/`role`/`password` para un usuario (solo admin).

- `health_handler.go`  

  - `GET /health`: healthcheck simple.

- `jwt_middleware.go`  

  - `JWTAuth(secret)`: middleware que valida el header `Authorization: Bearer <token>`, mete `userId` y `role` en el contexto.
  - `AdminOnly()`: middleware que permite solo requests con `role == "admin"`.
  - `UserIDFromContext(ctx)`: helper para extraer el id del usuario autenticado.

- `movie_handler.go`  

  - `GET /movies/{id}`: obtiene una película por ID.
  - `GET /movies/search`: búsqueda paginada de películas por texto / filtros.

- `rating_handler.go`  

  ADMIN:
  - `POST /users/{id}/ratings`
  - `GET /users/{id}/ratings`

  USER autenticado:
  - `POST /me/ratings`
  - `GET /me/ratings`

- `recommend_handler.go`  

  ADMIN:
  - `GET /users/{id}/recommendations`
  - `GET /users/{id}/ws/recommendations` (WebSocket)

  USER autenticado:
  - `GET /me/recommendations`

---

## 3. API principal (`cmd/api/main.go`)

El `main.go` de la API:

- Carga configuración con `config.Load()`.
- Inicializa Mongo (`db.InitMongo(cfg)`) y Redis (`cache.InitRedis(cfg)`).
- Crea repositorios: `UserRepository`, `MovieRepository`, `RatingRepository`, `RecommendationRepository`, `SimilarityRepository`.
- Lee direcciones de nodos ML desde `ML_NODE_ADDRS` (o usa fallback `mlnode1..4:9001`).
- Crea servicios: `AuthService`, `MovieService`, `RatingService`, `RecommendService`.
- Crea handlers.
- Configura el router `chi` con:
  - `Logger`, `Recoverer`.
  - Rutas públicas: `/health`, `/auth/*`, `/movies/*`.
  - Rutas protegidas con JWT: `/me/*` y `/users/*`.
  - Swagger: `/swagger/*`.
- Levanta el servidor en `HTTP_PORT`.

---

## 4. Autenticación y autorización (JWT + roles)

### 4.1. Token

En `auth_service.go`:

- `Register` crea usuario con rol (`admin` o `user`).
- `Login` genera un JWT con claims tipo:

    {
      "sub": 4,
      "role": "admin",
      "exp": 1763999999
    }

Respuesta del login:

    {
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9....",
      "userId": 4,
      "role": "admin"
    }

### 4.2. Middleware `JWTAuth`

En `jwt_middleware.go`:

    ctx := context.WithValue(r.Context(), CtxUserID, int(subVal))
    ctx = context.WithValue(ctx, CtxUserRole, role)

`UserIDFromContext(r.Context())` devuelve el id.

### 4.3. Middleware `AdminOnly`

Lee `CtxUserRole`. Si no es `"admin"` → `403 Forbidden`.

### 4.4. Rutas y reglas

    authMw := handler.JWTAuth(cfg.JWTSecret)

    r.Group(func(r chi.Router) {
        r.Use(authMw)

        r.Route("/me", func(r chi.Router) {
            r.Get("/ratings", ratingH.GetMyRatings)
            r.Post("/ratings", ratingH.PostMyRating)
            r.Get("/recommendations", recH.GetMyRecommendations)
        })

        r.Group(func(r chi.Router) {
            r.Use(handler.AdminOnly())

            r.Put("/users/{id}/update", authH.UpdateUser)

            r.Route("/users/{id}", func(r chi.Router) {
                r.Get("/ratings", ratingH.GetRatings)
                r.Post("/ratings", ratingH.PostRating)
                r.Get("/recommendations", recH.GetRecommendations)
                r.Get("/ws/recommendations", recH.GetRecommendationsWS)
            })
        })
    })

**Resumen rápido:**

- Rol `user`: puede usar `/me/*`.
- Rol `admin`: puede usar `/me/*` y además `/users/{id}/*`.

---

## 5. Recomendaciones y uso de Redis

### 5.1. `GET /me/recommendations` (usuario autenticado)

El cliente llama:

    GET /me/recommendations?k=20&refresh=false
    Authorization: Bearer <token>

Flujo:

1. `JWTAuth` mete `userId` en el contexto.
2. El handler llama a `RecommendService.Recommend`.
3. `RecommendService`:
   - Construye clave de caché `rec:user:<id>:k:<k>`.
   - Busca en Redis:
     - Si existe → devuelve directamente.
     - Si no existe o `refresh=true`:
       - Llama al cliente cluster → `mlnode1..4`.
       - Cada nodo lee ratings/similaridades de Mongo y devuelve `RecItem` (`movieId`, `score`, etc.).
       - Fusiona y ordena resultados, toma top-`k`.
       - Guarda en Redis.
4. El handler responde JSON con la lista de recomendaciones.

### 5.2. `GET /users/{id}/ws/recommendations` (admin + WebSocket)

Admin abre WebSocket:

    GET /users/{id}/ws/recommendations?k=20
    Authorization: Bearer <token_admin>

`RecommendHandler.GetRecommendationsWS`:

- Abre WebSocket.
- Envía mensaje inicial:

      { "type": "start", "msg": "Conexión WS abierta, iniciando cálculo…" }

- Envía mensajes de progreso por shard/nodo.
- Llama a `RecommendService.Recommend`.
- Envía mensaje final:

      {
        "type": "recommendations",
        "userId": 123,
        "items": [ ... ],
        "generatedAt": "2025-11-23T00:00:00Z"
      }

---

## 6. Módulo de ratings

### 6.1. Endpoints admin

- `POST /users/{id}/ratings`  

  Body de ejemplo:

      { "movieId": 123, "rating": 4.5 }

- `GET /users/{id}/ratings`  
  Retorna ratings de `{id}`.

### 6.2. Endpoints usuario autenticado

- `POST /me/ratings`
- `GET /me/ratings`

### 6.3. Persistencia

`rating_repo.go` guarda documentos como:

    {
      "userId": 123,
      "movieId": 456,
      "rating": 4.5,
      "timestamp": 1711234567
    }

Con `GetByUser` y `GetAllByUser` para lectura.

---

## 7. Nodos ML (`cmd/mlnode`)

### 7.1. Dockerfile de los nodos

    FROM golang:1.22-alpine AS builder
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    WORKDIR /app/cmd/mlnode
    RUN go build -o /mlnode

    FROM alpine:3.19
    WORKDIR /app
    COPY --from=builder /mlnode /app/mlnode
    EXPOSE 9001
    CMD ["/app/mlnode"]

### 7.2. Definición en `docker-compose.yml`

    mlnode1:
      build:
        context: .
        dockerfile: ./cmd/mlnode/Dockerfile
      container_name: pc4-mlnode1
      environment:
        ML_NODE_ADDR: ":9001"
        NODE_ID: "1"
        MONGO_URI: mongodb://root:example@mongo:27017
        MONGO_DB: pc4_movies
      networks:
        - pc4-net
    # idem mlnode2, mlnode3, mlnode4 con NODE_ID distinto

Cada nodo:

- Escucha en `:9001`.
- Usa `NODE_ID` para identificar su shard.
- Consulta Mongo y responde recomendaciones parciales.

---

## 8. Docker & `docker-compose`

El `docker-compose.yml` define:

- **mongo**
  - Imagen `mongo:7`
  - Puerto `27017`
  - Volumen `mongo_data`

- **mongo-express**
  - UI en `http://localhost:8081`

- **redis**
  - Imagen `redis:7`
  - Puerto `6379`
  - Volumen `redis_data`

- **api**
  - build usando `cmd/api/Dockerfile`
  - Expone `8080`
  - Variables:

    - `MONGO_URI`, `MONGO_DB`
    - `REDIS_ADDR`
    - `JWT_SECRET`
    - `HTTP_PORT`
    - `ML_NODE_ADDRS`

- **mlnode1..4**
  - build usando `cmd/mlnode/Dockerfile`
  - Comparten red `pc4-net`.

### 8.1. Dockerfile de la API

    FROM golang:1.22-alpine AS builder

    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    WORKDIR /app/cmd/api
    RUN go build -o /api

    FROM alpine:3.19
    WORKDIR /app
    COPY --from=builder /api /app/api
    ENV HTTP_PORT=8080
    EXPOSE 8080
    CMD ["/app/api"]

---

## 9. Configuración (.env vs docker-compose)

En `internal/config/config.go`:

    type Config struct {
        MongoURI  string
        MongoDB   string
        RedisAddr string
        RedisPass string
        JWTSecret string
        HTTPPort  string
    }

Ejemplo `.env` local:

    MONGO_URI=mongodb://root:example@localhost:27017
    MONGO_DB=pc4_movies
    REDIS_ADDR=localhost:6379
    REDIS_PASSWORD=
    JWT_SECRET=supersecret_jwt_para_pc4
    HTTP_PORT=8080
    ML_NODE_ADDRS=localhost:9001,localhost:9002,localhost:9003,localhost:9004

En Docker, se sobrescriben con los valores del `docker-compose.yml`.

---

## 10. Swagger y pruebas

En código:

    r.Get("/swagger/*", httpSwagger.WrapHandler)

URL:

- `http://localhost:8080/swagger/index.html`

Flujo típico:

1. `POST /auth/register` (crear admin o user).
2. `POST /auth/login` → copiar token.
3. En Swagger: botón **Authorize** → escribir `Bearer <token>`.
4. Probar `/me/*`.
5. Probar `/users/{id}/*` con rol admin (y ver `403` si el rol no es correcto).

---

## 11. Cómo levantar todo el proyecto

    # 1. Levantar todo
    docker compose up -d --build

    # 2. Ver logs API
    docker logs -f pc4-api

    # 3. Swagger
    #   http://localhost:8080/swagger/index.html

    # 4. Mongo Express (opcional)
    #   http://localhost:8081

Para apagar:

    docker compose down

Los volúmenes `mongo_data` y `redis_data` pueden conservar la data.

---

## 12. Flujo completo del proyecto (de punta a punta)

### 12.1. Diagrama de flujo general

    Cliente (Swagger / Front)
      └─HTTP (/auth, /me, /users)
           ↓
        API pc4-api
        - Handlers HTTP
        - Middlewares JWT/roles
           ↓
      ┌──────────────┬─────────────────────────────┐
      │              │                             │
    MongoDB      RatingService                RecommendService
    users/auth   (lectura/escritura           (recomendaciones
    movies/base   de ratings)                  top-N)
      │              │                             │
      │             CRUD                           │
      │                                            ↓
                                      Redis cache (rec:user:<id>:k)
                                      │     cache hit / miss
                                      ↓
                                 Cluster client (mlnode1..4)
                                      │  HTTP interno
                                      ↓
                         mlnode1..4 → consumen MongoDB (ratings, sims, movies)
                                      calculan recomendaciones parciales por shard
                                      devuelven lista de (movieId, score)

- El cliente interactúa siempre con la API (no habla directo con Mongo/Redis/nodos).
- La API usa MongoDB para autenticar usuarios y persistir películas, ratings y similitudes.
- Cuando se pide `/me/recommendations`, la API pasa por `RecommendService`, que:
  - Revisa primero Redis.
  - Si no hay datos, dispara el cálculo distribuido enviando requests al cluster de nodos ML.
  - Los nodos ML consultan MongoDB, calculan recomendaciones parciales y las devuelven.
  - El servicio combina, guarda en Redis y responde al cliente.
- Para ratings, la API pasa por `RatingService`, que escribe y lee directo de MongoDB.

### 12.2. Resumen del flujo (paso a paso)

1. Infraestructura: `docker compose up -d --build` levanta Mongo, Redis, API y 4 nodos ML.
2. Documentación: se abre Swagger y se inspeccionan endpoints.
3. Usuarios y login:
   - Se registra un usuario (`/auth/register`).
   - Se hace login (`/auth/login`) y se obtiene el JWT.
4. Reglas por rol:
   - Se configura `Bearer <token>` en Swagger.
   - Se prueba `/me/*` con usuario normal.
   - Se prueba `/users/{id}/*` con admin y se valida el `403` cuando el rol no es correcto.
5. Ratings:
   - Usuarios califican películas (`/me/ratings`).
   - Los datos quedan en Mongo (`ratings`).
6. Recomendaciones (HTTP):
   - Usuario llama a `/me/recommendations`.
   - `RecommendService` consulta Redis → (si miss) nodos ML → Mongo.
   - Combina resultados, guarda en caché y responde al cliente.
7. Recomendaciones (WebSocket):
   - Admin abre `/users/{id}/ws/recommendations`.
   - Recibe mensajes `start`, progreso por nodo y `recommendations` final.
   - Se evidencia el uso de los 4 nodos de cómputo.
8. Monitoreo de datos:
   - Con `mongo-express` se revisan colecciones (`users`, `ratings`, `movies`, `recommendations`).
   - Opcionalmente, con `redis-cli` se inspeccionan claves de caché.
9. Cierre:
   - `docker compose down` apaga todo.
   - Los volúmenes `mongo_data` y `redis_data` pueden conservar la data.

---

## 13. Resumen final

Este proyecto demuestra:

- Un sistema de recomendación distribuido apoyado en múltiples nodos ML.
- Una API REST + WebSocket en Go, organizada por capas (`handler → service → repository`).
- Integración con MongoDB para persistencia y Redis para caché.
- Seguridad con JWT y roles (`user`, `admin`), reflejada en `/me/*` y `/users/{id}/*`.
- Infraestructura reproducible con Docker y `docker-compose`.
- Documentación completa mediante Swagger.