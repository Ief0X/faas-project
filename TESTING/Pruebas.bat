@echo off

echo Current directory: %CD%
docker compose up --build -d
timeout /t 2

echo _________________________________________________________________________
echo Creando usuario de prueba...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Creando usuario 2...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser2\",\"password\":\"testpass2\"}"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Iniciando sesión como testuser y recuperando token...
curl -X POST http://localhost:9080/login ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}" > login_response.json

if not exist login_response.json (
    echo "Error: login_response.json was not created!"
    exit /b
)
echo _________________________________________________________________________

FOR /F "delims=" %%i IN ('powershell -NoProfile -Command ^
    "(Get-Content -Path 'login_response.json' | ConvertFrom-Json).token"') DO SET TOKEN=%%i

if "%TOKEN%"=="" (
    echo "Error: Token could not be retrieved!"
    exit /b
)

DEL login_response.json
echo _________________________________________________________________________

cd TESTING
echo Current directory: %CD%
docker build -t functionbyuser .
timeout /t 2

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser antes de la eliminación...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Registrando función para testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser antes de la eliminación...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser desde testuser2...
curl -X GET "http://localhost:9080/functions?username=testuser2" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Ejecutando testfunction1...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"happy\"}" ^
     http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Ejecutando testfunction1 desde testuser2...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"happy\"}" ^
     http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Eliminando testfunction1...
curl -X DELETE http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser después de la eliminación...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Registrando función para testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" ^
     http://localhost:9080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Obteniendo todas las funciones asociadas con testuser después de la eliminación...
curl -X GET "http://localhost:9080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Eliminando testfunction1...
curl -X DELETE http://localhost:9080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo Prueba completada.
cd ..
pause
