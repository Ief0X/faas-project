# faas-project

docker compose up --build 

curl -X POST http://localhost:9080/register -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

curl -X POST http://localhost:9080/login -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\"}"

cd TESTING

docker build -t emociones .

curl -X POST -H "Content-Type: application/json" -d "{\"name\": \"emociones\", \"image\": \"emociones\"}" http://localhost:8080/function

curl -X POST -H "Content-Type: application/json" -d "{\"param\": \"happy\"}" http://localhost:8080/function/emociones

curl -X DELETE http://localhost:8080/function/emociones


