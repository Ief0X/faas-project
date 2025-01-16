@echo off

echo Current directory: %CD%
docker compose up --build -d
timeout /t 2

echo Usuario de prueba...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

echo Token de autenticación...
curl -X POST http://localhost:9080/login ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}" > login_response.json

FOR /F "delims=" %%i IN ('powershell -NoProfile -Command ^
    "(Get-Content -Path 'login_response.json' | ConvertFrom-Json).token"') DO SET TOKEN=%%i

DEL login_response.json

cd TESTING
docker build -t functionbyuser .
cd ..

echo Registrando múltiples funciones de prueba...
for /l %%i in (1,1,6) do (
    curl -X POST -H "Content-Type: application/json" ^
         -d "{\"name\": \"testfunction%%i\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" ^
         http://localhost:9080/function ^
         -H "Authorization: Bearer %TOKEN%"
)

: Se ejecuta en una CMD que se abre rápidamente, no os asusteis.
timeout /t 2
: Esto no funciona del todo
echo Ejecutando misma función simultáneamente...
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"test1\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"testdfhbfgbn2\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"testdfggdr3\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"tesdrgdrt4\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"test5ddrg\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
timeout /t 8

echo Ejecutando diferentes funciones simultáneamente...
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"test1\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"testdfhbfgbn2\"}" http://localhost:9080/function/testfunction2 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"testdfggdr3\"}" http://localhost:9080/function/testfunction3 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"tesdrgdrt4\"}" http://localhost:9080/function/testfunction4 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"test5ddrg\"}" http://localhost:9080/function/testfunction5 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"test6ddrg\"}" http://localhost:9080/function/testfunction6 -H "Authorization: Bearer %TOKEN%""
timeout /t 8

echo Limpieza...
for /l %%i in (1,1,6) do (
    curl -X DELETE http://localhost:9080/function/testfunction%%i ^
         -H "Authorization: Bearer %TOKEN%"
    timeout /t 1
)

echo Prueba completada.
pause