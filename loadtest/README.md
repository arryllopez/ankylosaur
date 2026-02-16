# Load Testing ankyloGo

This directory contains load testing infrastructure for measuring ankyloGo's performance.

## Prerequisites

Install `hey` (HTTP load testing tool):
```bash
go install github.com/rakyll/hey@latest
```

## Running the tests

### Option 1: Automated script (Linux/Mac)
```bash
chmod +x loadtest/loadtest.sh
./loadtest/loadtest.sh
```

### Option 2: Manual testing (Windows/Any)

1. Start the test server:
```bash
go run loadtest/server.go
```

2. In another terminal, run baseline test:
```bash
hey -n 100000 -c 100 http://localhost:8080/baseline
```

3. Run middleware test:
```bash
hey -n 100000 -c 100 http://localhost:8080/with-middleware
```

## Understanding the results

`hey` will output metrics like:
- **Requests/sec**: Throughput (how many requests per second)
- **Average latency**: Average response time
- **Fastest/Slowest**: Min/max latency

### Calculating overhead

Overhead is the additional latency added by the middleware:
```
Overhead = Middleware Average Latency - Baseline Average Latency
```

### Example output interpretation

```
Baseline:
  Requests/sec: 45000
  Average: 2.2 ms

With Middleware:
  Requests/sec: 40000
  Average: 2.5 ms

Overhead: 2.5 - 2.2 = 0.3 ms
```

Resume bullet example:
> "Sustained 40,000+ req/sec under load testing with <1ms overhead"

## Tuning for different scenarios

Edit `loadtest/server.go` to adjust:
- Rate limit configurations
- Enable/disable Kafka publishing
- Add Redis backend
- Test different endpoints

Edit `hey` parameters:
- `-n`: Total number of requests (default: 100000)
- `-c`: Concurrency (number of workers, default: 100)
- `-q`: Rate limit for requests per second (optional)
- `-z`: Duration instead of request count (e.g., `-z 30s`)

Example for 30-second test:
```bash
hey -z 30s -c 100 http://localhost:8080/with-middleware
```
