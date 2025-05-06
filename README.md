# Go Proxy Worker Boilerplate

This is a concurrent worker system in Go that distributes tasks across multiple proxxies. It automatically manages proxy reuse, handles rate-limited or banned proxies, and allows for easy task customization.

## Features

- Efficient task queue with retry handling
- Worker pool using proxies from `proxies.txt`
- Automatic proxy rotation and lifecycle management
- Supports handling of rate limits and permanently dead proxies
- Easy plug-in of your own logic via `process.go`

## Usage

### 1. Prepare Input Files

- `proxies.txt`: List of proxies, one per line.
- `input.txt`: Input data, one task per line, each line becomes `Task.Data`.

### 2. Customize Task Logic

Edit `process.go` and implement your custom task logic using:
```go
func process(task Task, proxy string) error
```

You receive:

- `task.Data` - one line from `input.txt`
- `proxy` - one proxy from `proxies.txt`

Use the following utility functions inside `process.go` to handle proxy issues:

#### Available Proxy Management Functions

```go
removeProxy(proxy string)
```
> Permanently removes a proxy (e.g.m dead or banned).

```go
releaseProxyLater(proxy string, delay time.Duration)
```
> Temporarily removes a proxy, re-adds it after a delay (e.g., rate-limited).

### 3. Run the program
```go
go run .
```

You'll be prompted to enter the number of conurrent workers. It will automatically limit to the number of available proxies.

## Worker Error Behavior

By **default**, if `process()` returns an error:

- The current worker **exits**.
- A new worker is spawned with a **new proxy**.
- The task is added to the **retry queue**.

You can customize this behaviour in `worker()` inside `main.go`.

For example:

- Skip to the next task instead of stopping the worker
- Remove a proxy only for specific errors
- Apply different retry strategies

This gives you full control over how errors and proxy health are handled.

## File Structure
```text
├── main.go        # Main app: task management, proxy logic, worker orchestration
├── process.go     # Your custom task logic
├── proxies.txt    # List of proxy addresses
└── input.txt      # List of input tasks
```
