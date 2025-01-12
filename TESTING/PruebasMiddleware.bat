@echo off
docker compose up --build -d
timeout /t 2

echo _________________________________________________________________________
echo Creating user testuser...
curl -X POST http://localhost:9080/register ^
     -H "Content-Type: application/json" ^
     -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Logging in as testuser and retrieving token...
curl -X POST http://localhost:8080/login ^
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

docker build -t functionbyuser .
timeout /t 2

echo _________________________________________________________________________
echo Registering function for testuser...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"id\": \"1\", \"name\": \"testfunction1\", \"ownerId\": \"testuser\", \"image\": \"functionbyuser\"}" ^
     http://localhost:8080/function ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Getting all functions associated with testuser before deletion...
curl -X GET "http://localhost:8080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Executing testfunction1...
curl -X POST -H "Content-Type: application/json" ^
     -d "{\"param\": \"input\"}" ^
     http://localhost:8080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Deleting testfunction1...
curl -X DELETE http://localhost:8080/function/testfunction1 ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Getting all functions associated with testuser after deletion...
curl -X GET "http://localhost:8080/functions?username=testuser" ^
     -H "Authorization: Bearer %TOKEN%"
echo _________________________________________________________________________

echo Script completed.
pause
