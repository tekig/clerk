# Clerk
A lightweight key-value store that writes data to S3 with infrequent read support.

## Feature
- Stores data in S3 (or compatible systems like MinIO)
- Optimized for frequent writes and rare reads

## Recorder
Recorder is responsible for collecting data and writing it to S3. It can be run with multiple replicas.

## Searcher
Searcher looks up values by UUID in S3. It must have access to all Recorder replicas.

## Hammer
Hammer is a load-testing tool for stressing the Recorders and measuring their performance.

## Simple run
First, you need to create a network to connect the components to each other.
```bash
docker network create clerk
```

We create a recorder to which events will be sent for saving in the future. There may be several pods, the main thing is to indicate them in the other components.
```bash
docker run -d --name recorder \
    --volume $(pwd)/config/recorder.yaml:/app/config.yaml \
    --network clerk \
    ghcr.io/tekig/clerk:main \
    recorder --config /app/config.yaml
```

Creates a searcher sub-folder that will search within already created blocks and blocks that have not yet been written.
```bash
docker run -d --name searcher \
    --volume $(pwd)/config/searcher.yaml:/app/config.yaml \
    --network clerk \
    -p 8080:8080 \
    ghcr.io/tekig/clerk:main \
    searcher --config /app/config.yaml
```

To fake data recording, we will run hammer for a short time. CTRL+C to interrupt.
```bash
docker run --name hammer \
    --network clerk \
    ghcr.io/tekig/clerk:main \
    hammer --host recorder:50051
```

Select a random UUID from the log to search and go to the localhost:8080 interface. After entering the UUID in the search, you will see randomly recorded data. You may get a `not found` error for recently written events. This is due to the snappy buffer, write more data to find the latest data.
