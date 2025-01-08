# FaaS Project

## Equipo de Desarrollo

| Nombre | 
|--------|
| Fernando Garcia Barra |
| Jefferson Paul Caiza Jami |
| Pablo Granell Robles |

## Guía de Inicio Rápido

docker compose up --build 

curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

cd TESTING

docker build -t emociones .

curl -X POST -H "Content-Type: application/json" -d "{\"name\": \"emociones\", \"image\": \"emociones\"}" http://localhost:8080/function

curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"happy\"}" http://localhost:8080/function/emociones

curl -X DELETE http://localhost:8080/function/emociones

## IMPORTANTE

Para cerrar el contenedor correctamente hay que parar el contenedor desde docker. (Por ejemplo el boton de parar desde docker desktop)

La utilizacion de CTRL+C causa que se corrompa el archivo de registro de logs de etcd.


