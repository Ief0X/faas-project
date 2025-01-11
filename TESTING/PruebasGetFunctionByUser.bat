@echo off
docker compose up --build -d
timeout /t 2

echo _________________________________________________________________________
echo Creando usuario testuser...
curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Iniciando sesion con testuser...
curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

docker build -t functionbyuser .
timeout /t 2

echo _________________________________________________________________________
echo Registrando funcion para testuser...
curl -X POST -H "Content-Type: application/json" -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" http://localhost:8080/function
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Registrando segunda funcion para testuser...
curl -X POST -H "Content-Type: application/json" -d "{\"id\": \"2\", \"name\": \"testfunction2\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" http://localhost:8080/function
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Registrando tercera funcion para testuser...
curl -X POST -H "Content-Type: application/json" -d "{\"id\": \"3\", \"name\": \"testfunction3\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" http://localhost:8080/function
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas a testuser antes de eliminar...
curl -X GET "http://localhost:8080/functions?username=testuser"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Ejecutando testfunction1...
curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"input\"}" http://localhost:8080/function/testfunction1
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Ejecutando testfunction2...
curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"input\"}" http://localhost:8080/function/testfunction2
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Eliminando testfunction1...
curl -X DELETE http://localhost:8080/function/testfunction1
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Eliminando testfunction2...
curl -X DELETE http://localhost:8080/function/testfunction2
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas a testuser despues de eliminar...
curl -X GET "http://localhost:8080/functions?username=testuser"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Script completado.
pause