# faas-project

docker compose up --build 

curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

curl -X POST http://localhost:9080/function -H "Content-Type: application/json" -d "{\"name\":\"test\",\"image\":\"alpine\"}"

curl -X POST http://localhost:9080/function/test -H "Content-Type: application/json" -d "{\"param\":\"hola\"}"

curl -X DELETE http://localhost:9080/function/test