Click Counter Service (MongoDB) — Go 1.24

Docker turn on/turn off:
docker-compose up -d / docker-compose down

Single click on the banner:
curl http://localhost:3000/counter/1

Multiple clicks on a banner:
seq 1 200 | xargs -n1 -P50 curl -s http://localhost:3000/counter/1

Getting general statistics:
curl -X POST -H "Content-Type: application/json" \
-d '{"from":"2025-10-22T22:40:00+03:00","to":"2025-10-22T23:20:00+03:00"}' \
http://localhost:3000/stats

Getting banner statistics:
curl -X POST -H "Content-Type: application/json" \
-d '{"from":"2025-10-22T23:30:00+03:00","to":"2025-10-22T23:40:00+03:00"}' \
http://localhost:3000/stats/1