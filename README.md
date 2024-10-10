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

Para terminar los procesos, ejecutar
```bash
make compose-down
```
