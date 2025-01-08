@echo off
docker compose up --build -d
timeout /t 2

echo _________________________________________________________________________
curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

echo _________________________________________________________________________
curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"
echo _________________________________________________________________________

docker build -t emociones .
timeout /t 2

echo _________________________________________________________________________
curl -X POST -H "Content-Type: application/json" -d "{\"name\": \"emociones\", \"image\": \"emociones\"}" http://localhost:8080/function
echo _________________________________________________________________________

echo _________________________________________________________________________
curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"happy\"}" http://localhost:8080/function/emociones
echo _________________________________________________________________________

echo _________________________________________________________________________
curl -X DELETE http://localhost:8080/function/emociones
echo _________________________________________________________________________

echo _________________________________________________________________________
echo Script completado.
pause