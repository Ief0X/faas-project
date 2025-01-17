@echo off

echo Directorio actual: %CD%
docker compose up --build -d
timeout /t 2

echo _________________________________________________________________________
echo Creando usuario de prueba...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________
echo Creando usuario 2...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser2\",\"password\":\"testpass2\"}"
echo _________________________________________________________________________
echo Iniciando sesion como testuser y recuperando token...
curl -X POST http://localhost:9080/login ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}" > login_response.json

if not exist login_response.json (
    echo "Error: login_response.json was not created!"
    exit /b
)
FOR /F "delims=" %%i IN ('powershell -NoProfile -Command ^
    "(Get-Content -Path 'login_response.json' | ConvertFrom-Json).token"') DO SET TOKEN=%%i

if "%TOKEN%"=="" (
    echo "Error: Token no encontrado"
    exit /b
)
DEL login_response.json
echo _________________________________________________________________________
echo No hay funciones asociadas con testuser...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Registrando funcion para testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"pablogranell/traductor\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Registrando funcion para testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"2\", \"name\": \"testfunction2\", \"ownerId\": \"testuser\", \"image\": \"pablogranell/emociones\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Registrando funcion para testuser2...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction2\", \"ownerId\": \"testuser2\", \"image\": \"pablogranell/emociones\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser desde testuser2...
curl -X GET "http://localhost:9080/functions?username=testuser2" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Ejecutando testfunction1...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"happy\"}" ^
     http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Ejecutando testfunction2 desde testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"happy\"}" ^
     http://localhost:9080/function/testfunction2 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Ejecutando testfunction2 desde testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"sad\"}" ^
     http://localhost:9080/function/testfunction2 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Eliminando testfunction1...
curl -X DELETE http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Eliminando testfunction2 desde testuser...
curl -X DELETE http://localhost:9080/function/testfunction2 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser despues de la eliminacion...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Registrando funcion para testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"emociones\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser despues de la eliminacion...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________
echo Eliminando testfunction1...
curl -X DELETE http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo Prueba completada.
pause
