# FaaS Project

## Equipo de Desarrollo

| Nombre 					 | 
|----------------------------|
| Fernando Garcia Barra      |
| Jefferson Paul Caiza Jami  |
| Pablo Granell Robles       |


## IMPORTANTE

Hemos detectado un problema con etcd, que provoca la corrompción y bloqueo del funcionamiento del sistema al corromperse el archivo de registro de logs.
Este WAL se corrompe cuando se cierra el contenedor de etcd incorrectamente y la solución que hemos encontrado es detener el contenedor de etcd desde Docker Desktop.

Si al ejecutar el docker-compose hubiera un error fatal en los registros de etcd, es necesario revertir los cambios en WAL y DB, por ejemplo, con un git stash.

Lo hemos probado en Windows con WSL2 y funciona correctamente.

## IMPORTANTE 2

En caso de ejecutarse en linux será necesario ejecutar previamente el siguiente comando para que la carpeta sea visible para los contenedores 
```
sudo chmod -R 777 etcd/
```
Por limitaciones del contenedor etc será necesario ejecutar este comando cada vez que se vaya a arrancar, ya que internamente puede modificar los permisos

## Guía de Inicio Rápido

```
	docker compose up --build -d
```

```
curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"Usuario1\",\"password\":\"test1\"}"
```

```
curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"Usuario1\",\"password\":\"test1\"}"
```

Copiar el token de la respuesta y reemplazar <TOKEN> por el token en los siguientes comandos

```
curl -X POST -H "Content-Type: application/json" -d "{\"id\": \"1\", \"name\": \"Funcion1\", \"ownerId\": \"Usuario1\", \"image\": \"pablogranell/emociones\"}" http://localhost:9080/function -H "Authorization: Bearer <TOKEN>"
```

```
curl -X GET "http://localhost:9080/functions?username=Usuario1" -H "Authorization: Bearer <TOKEN>"
```

```
curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"happy\"}" http://localhost:9080/function/Funcion1 -H "Authorization: Bearer <TOKEN>"
```

```
curl -X DELETE http://localhost:9080/function/Funcion1 -H "Authorization: Bearer <TOKEN>"
```

```
curl -X GET http://localhost:9080/functions?username=Usuario1 -H "Authorization: Bearer <TOKEN>"
```



