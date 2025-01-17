@echo off

echo Current directory: %CD%
docker compose up --build -d
timeout /t 2

echo Usuario de prueba...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

echo Token de autenticacion...
curl -X POST http://localhost:9080/login ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}" > login_response.json

FOR /F "delims=" %%i IN ('powershell -NoProfile -Command ^
    "(Get-Content -Path 'login_response.json' | ConvertFrom-Json).token"') DO SET TOKEN=%%i

DEL login_response.json

echo Registrando multiples funciones de prueba...
for /l %%i in (1,1,6) do (
    curl -X POST -H "Content-Type: application/json" ^
         -d "{\"name\": \"testfunction%%i\", \"ownerId\": \"testuser\", \"image\": \"pablogranell/emociones\"}" ^
         http://localhost:9080/function ^
         -H "Authorization: Bearer %TOKEN%"
)

echo Ejecutando misma funcion simultaneamente...
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"happy\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"sad\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"rage\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"bubbly\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"sadness\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
timeout /t 2

echo Ejecutando diferentes funciones simultaneamente...
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"envy\"}" http://localhost:9080/function/testfunction1 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"tired\"}" http://localhost:9080/function/testfunction2 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"angry\"}" http://localhost:9080/function/testfunction3 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"clown\"}" http://localhost:9080/function/testfunction4 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"drunk\"}" http://localhost:9080/function/testfunction5 -H "Authorization: Bearer %TOKEN%""
start cmd /k "curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"sober\"}" http://localhost:9080/function/testfunction6 -H "Authorization: Bearer %TOKEN%""
timeout /t 10

echo Limpieza...
for /l %%i in (1,1,6) do (
    curl -X DELETE http://localhost:9080/function/testfunction%%i ^
         -H "Authorization: Bearer %TOKEN%"
)

echo Prueba completada.
pause