# faas-project

docker compose up --build 

curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"