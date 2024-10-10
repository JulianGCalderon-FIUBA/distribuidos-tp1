# Distribuidos - TP1

## Configurar cantidad de nodos por query
Para actualizar el archivo `compose.yaml` con la cantidad de nodos deseada, modificar las constantes del script `cmd/compose/main.go` y luego ejecutar el comando:
```bash
make write-compose
```

## Ejecucion con Docker

Para levantar los procesos, ejecutar:
```bash
make compose-up
```

Para ver los logs del sistema, ejecutar:
```bash
make compose-logs
```

Para terminar los procesos, ejecutar
```bash
make compose-down
```

Para ver las diferencias entre las librerías de lenguaje de Go y Python, ejecutar los scripts:
```bash
go run ./cmd/detect-language/main.go 
python3 ./cmd/detect-language/main.py
```
Aclaración: es conveniente correr estos scripts en paralelo ya que tardan bastante tiempo.

Y luego ejecutar:
```bash
./cmd/detect-language/diff.sh
```

