.PHONY: proto build-server build-client build run-node1 run-node2 run-node3 run-servers run-client clean test kill-servers help

# Variables
PROTO_DIR=proto
PB_DIR=pb
BIN_DIR=bin
SERVER_BIN=$(BIN_DIR)/server
CLIENT_BIN=$(BIN_DIR)/client
GOPATH_BIN ?= $(shell go env GOPATH)/bin

# Crear directorios necesarios
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(PB_DIR):
	mkdir -p $(PB_DIR)

# Generar código desde proto
proto: $(PB_DIR)
	@echo "Generando código desde archivos proto..."
	@echo "Usando plugins de $(GOPATH_BIN)"
	protoc --plugin=protoc-gen-go=$(GOPATH_BIN)/protoc-gen-go \
	       --plugin=protoc-gen-go-grpc=$(GOPATH_BIN)/protoc-gen-go-grpc \
	       --go_out=. \
	       --go-grpc_out=. \
	       $(PROTO_DIR)/vector.proto

# Compilar servidor
build-server: $(BIN_DIR)
	@echo "Compilando servidor..."
	go build -o $(SERVER_BIN) server/main.go

# Compilar cliente
build-client: $(BIN_DIR)
	@echo "Compilando cliente..."
	go build -o $(CLIENT_BIN) client/main.go

# Compilar todo
build: proto build-server build-client
	@echo "✅ Compilación completa"

# Ejecutar nodo 1
run-node1: build-server
	$(SERVER_BIN) -id=1 -port=50051 -pfail=0.1 -pcrash=0.05 -rinit=1000

# Ejecutar nodo 2
run-node2: build-server
	$(SERVER_BIN) -id=2 -port=50052 -pfail=0.15 -pcrash=0.08 -rinit=1000

# Ejecutar nodo 3
run-node3: build-server
	$(SERVER_BIN) -id=3 -port=50053 -pfail=0.2 -pcrash=0.1 -rinit=1000

# Ejecutar todos los servidores en background
run-servers: build-server
	@echo "Iniciando 3 nodos servidores..."
	@$(SERVER_BIN) -id=1 -port=50051 -pfail=0.1 -pcrash=0.05 -rinit=1000 > server1.log 2>&1 &
	@$(SERVER_BIN) -id=2 -port=50052 -pfail=0.15 -pcrash=0.08 -rinit=1000 > server2.log 2>&1 &
	@$(SERVER_BIN) -id=3 -port=50053 -pfail=0.2 -pcrash=0.1 -rinit=1000 > server3.log 2>&1 &
	@echo "Servidores iniciados. Logs en server1.log, server2.log, server3.log"
	@sleep 2

# Ejecutar cliente
run-client: build-client
	$(CLIENT_BIN) -n=20 -rinit=1000 \
		-pfail1=0.1 -pfail2=0.15 -pfail3=0.2 \
		-pcrash1=0.05 -pcrash2=0.08 -pcrash3=0.1

# Detener servidores
kill-servers:
	@echo "Deteniendo servidores..."
	@pkill -f "$(SERVER_BIN)" || true
	@echo "Servidores detenidos"

# Ejecutar prueba completa
test: clean build
	@echo "=== Iniciando prueba completa ==="
	@make run-servers
	@sleep 2
	@make run-client
	@make kill-servers
	@echo "=== Mostrando resultados ==="
	@cat output.txt
	@echo "=== Prueba completada ==="

# Limpiar archivos generados
clean:
	@echo "Limpiando archivos generados..."
	@rm -rf $(BIN_DIR) $(PB_DIR) output.txt *.log
	@-make kill-servers
	@echo "Limpieza completada"

# Instalar dependencias
deps:
	@echo "Instalando dependencias (forzando recompilación)..."
	go mod download
	go install -a -v google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install -a -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Dependencias instaladas"

# Ayuda
help:
	@echo "Sistema de Suma de Vectores con gRPC"
	@echo "====================================="
	@echo "Comandos disponibles:"
	@echo "  make deps        - Instalar dependencias necesarias"
	@echo "  make proto       - Generar código Go desde archivos .proto"
	@echo "  make build       - Compilar servidor y cliente"
	@echo "  make run-servers - Iniciar los 3 nodos servidores en background"
	@echo "  make run-client  - Ejecutar cliente con parámetros por defecto"
	@echo "  make test        - Ejecutar prueba completa del sistema"
	@echo "  make clean       - Limpiar archivos generados y detener servidores"
	@echo "  make kill-servers- Detener todos los servidores en ejecución"
	@echo ""
	@echo "Ejecución individual de nodos:"
	@echo "  make run-node1   - Ejecutar nodo 1 (puerto 50051)"
	@echo "  make run-node2   - Ejecutar nodo 2 (puerto 50052)"
	@echo "  make run-node3   - Ejecutar nodo 3 (puerto 50053)"