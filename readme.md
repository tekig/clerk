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
