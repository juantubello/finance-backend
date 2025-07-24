# Etapa 1: Build
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copiamos dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el resto del proyecto
COPY . .

# Compilar el binario para producción
RUN go build -o finance-backend main.go

# Etapa 2: Contenedor liviano para producción
FROM alpine:latest

# Instalamos certificados SSL (para HTTPS o llamadas externas)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copiamos el binario desde la etapa de build
COPY --from=builder /app/finance-backend .

# Copiamos la carpeta de configuración de base de datos (estructura inicial)
COPY db ./db

# Copiamos también el archivo .env si es necesario en tu app
COPY .env .env

# Exponemos el puerto (ajustar si usás otro)
EXPOSE 8080

# Ejecutamos la app
CMD ["./finance-backend"]
