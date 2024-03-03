go build -o ./build/tm-load-test ./cmd/tm-load-test/main.go
./build/tm-load-test \
    coordinator \
    --expect-workers 2 \
    --bind localhost:26670 \
    -c 1 -T 10 -r 1000 -s 250 \
    --broadcast-tx-method async \
    --endpoints ws://localhost:26657/websocket --stats-output result.csv