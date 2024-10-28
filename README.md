# Distribuidos - TP1

<!--toc:start-->
- [Configurar cantidad de nodos por query](#configurar-cantidad-de-nodos-por-query)
- [Ejecucion con Docker](#ejecucion-con-docker)
- [Comparacion de resultados](#comparacion-de-resultados)
- [Usar dataset reducido](#usar-dataset-reducido)
<!--toc:end-->

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

Luego ejecutar:
```bash
./cmd/detect-language/diff.sh
```

## Comparacion de resultados

Para comparar los resultados, primero hay que obtener los valores de referencia. Tenemos un script de Python que los resuelve, pero necesitamos usar el mismo detector de lenguaje, para asegurar que los resultados sean los mismo. Para eso, ejecutamos:
```bash
go run ./scripts/filter-english-negative/main.go .data/reviews.csv .data/reviews-english-negative.csv
```

Luego, resolvemos las consultas localmente, ejecutando:
```bash
python ./scripts/solve.py .data/ .py-results/
```
Este guardara los resultados correctos en `.py-results/`

Para compararlos, ejecutamos:
```bash
./scripts/compare.sh .results/ .py-results/
```

## Usar dataset reducido

Primero, tenemos que generar un dataset reducido de datos. Para eso, ejecutamos:
```bash
go run ./scripts/reduce/main.go
```

Despues, tenemos que modificar el compose para que use el dataset reducido. Tenemos que cambiar cual carpeta de los datos se bindea al contenedor del cliente. Se puede hacer manualmente, o automaticamente ejecutando:
```bash
sed -i 's|./.data:/.data|./.data-reduced:/.data|' ./scripts/compose/main.go # linux
sed -i '' 's|./.data:/.data|./.data-reduced:/.data|' ./scripts/compose/main.go # osx
```

Luego, regeneramos el compose:
```bash
make write-compose
```

Luego de ejecutar el sistema, podemos adaptar la seccion de [comparacion de resultados](#comparacion-de-resultados) para utilizar el dataset reducido.
