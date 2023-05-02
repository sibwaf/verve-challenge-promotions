# Challenge

We receive some records in a CSV file (example promotions.csv attached) every 30 minutes. We would like to store these objects in a way to be accessed by an endpoint.

Given an ID the endpoint should return the object, otherwise, return not found.

Eg:

`curl https://localhost:1321/promotions/172FFC14-D229-4C93-B06B-F48B8C095512`
```json
{ "id": "172FFC14-D229-4C93-B06B-F48B8C095512", "price": 9.68, "expiration_date": "2022-06-04 06:01:20" }
```

Additionally, consider:
* The .csv file could be very big (billions of entries) - how would your application perform?
* Every new file is immutable, that is, you should erase and write the whole storage;
* How would your application perform in peak periods (millions of requests per minute)?
* How would you operate this app in production (e.g. deployment, scaling, monitoring)?
* The application should be written in golang;
* Main deliverable is the code for the app including usage instructions, ideally in a repo/github gist.

# Solution

## Quick start

```shell
docker-compose up

# Upload data
curl --request POST \
     --url "http://localhost:1322/upload" \
     --header "Content-Type: multipart/form-data" \
     --form data=@promotions.csv

# OK
curl -v "http://localhost:1321/promotions/d018ef0b-dbd9-48f1-ac1a-eb4d90e57118"

# Not Found
curl -v "http://localhost:1321/promotions/test"

# Prometheus metrics
curl -v "http://localhost:1321/prometheus"
```

## Description

Our primary storage options:
1. Save the incoming file in a performant storage and manually build an index (ex. BTree) of ID -> position in file. Requires "reinventing the wheel", which is usually not a good idea.
2. Use a key-value database. Could be really performant, but requires possibly non-trivial working around the "erase and write the whole storage" requirement for atomic dataset switches to prevent outages and partial dataset access.
3. Use an SQL/NoSQL database.

In this solution we use MariaDB/MySQL as the storage. As soon as it's reading performance starts being the bottleneck we could (and should) scale it horizontally with read-only replicas. The writing performance bottleneck would most likely require vertical scaling, though we could try and cluster the data between multiple servers.

The atomic dataset switch problem is solved using two tables with the same schema - one of the tables holds the live data that gets served, and the other table is the "staging" one. On a dataset update their roles get switched.

The app itself is divided into two parts:
1. "updater" - imports data into the database and manages dataset switches. It's not scalable in the current implementation, but if it becomes a bottleneck we could rework it to support splitting the raw data file into multiple chunks and feed them into multiple database shards.
2. "api" - serves the API (i.e. the /promotions/{guid} URL in our case). Fully scalable.

This approach of splitting the responsibilities allows independent scaling and simple switching to "read-only mode" by scaling down all "updater" instances to 0. While the separate parts can be developed as completely separate projects, having them in the same application makes code sharing between them easier.

Under high load the app suffers from increased response latencies, though the main bottleneck in this case seems to be the database. On Ryzen 4500U, a single API node (with MaxOpenConns set to 128) with a single untuned MariaDB 10.7.8 instance in Docker is able to handle 10000 RPS on average (`bombardier -c 512 http://localhost:1321/promotions/d018ef0b-dbd9-48f1-ac1a-eb4d90e5711`) without maxing out CPU usage. It could also be beneficial to introduce a cache (ex. Redis) if some IDs are expected to be frequently requested. Having multiple read-only database replicas on standby and scaling up the app on demand should be enough to handle the usage peaks.

The app is completely stateless and can be easily deployed in a Kubernetes cluster with autoscaling for the API node. It includes a Prometheus endpoint available at `/prometheus` (ex. http://localhost:1321/prometheus) for monitoring, which exports metrics for HTTP requests/responses, resource consumption and database connection statistics. A Grafana dashboard (or any other compatible solution) can be setup in production along with alerts.
