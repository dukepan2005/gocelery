# gocelery

Go Client/Server for Celery Distributed Task Queue

[![Build Status](https://github.com/gocelery/gocelery/workflows/Go/badge.svg)](https://github.com/gocelery/gocelery/workflows/Go/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/gocelery/gocelery/badge.svg?branch=master)](https://coveralls.io/github/gocelery/gocelery?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gocelery/gocelery)](https://goreportcard.com/report/github.com/gocelery/gocelery)
[!["Open Issues"](https://img.shields.io/github/issues-raw/gocelery/gocelery.svg)](https://github.com/gocelery/gocelery/issues)
[!["Latest Release"](https://img.shields.io/github/release/gocelery/gocelery.svg)](https://github.com/gocelery/gocelery/releases/latest)
[![GoDoc](https://godoc.org/github.com/gocelery/gocelery?status.svg)](https://godoc.org/github.com/gocelery/gocelery)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/gocelery/gocelery/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgocelery%2Fgocelery.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgocelery%2Fgocelery?ref=badge_shield)

## Why?

Having been involved in several projects migrating servers from Python to Go, I have realized Go can improve performance of existing python web applications.
As Celery distributed tasks are often used in such web applications, this library allows you to both implement celery workers and submit celery tasks in Go.

You can also use this library as pure go distributed task queue.

## Go Celery Worker in Action

![demo](https://raw.githubusercontent.com/gocelery/gocelery/master/demo.gif)

## Supported Brokers/Backends

Now supporting both Redis and AMQP!!

* Redis (broker/backend)
* AMQP (broker/backend) - does not allow concurrent use of channels

## Celery Configuration

Celery must be configured to use **json** instead of default **pickle** encoding.
This is because Go currently has no stable support for decoding pickle objects.
Pass below configuration parameters to use **json**.

Starting from version 4.0, Celery uses message protocol version 2 as default value.
GoCelery now support message protocol version 2, so you must explicitly set `CELERY_TASK_PROTOCOL` to 1.

```python
CELERY_TASK_SERIALIZER='json',
CELERY_ACCEPT_CONTENT=['json'],  # Ignore other content
CELERY_RESULT_SERIALIZER='json',
CELERY_ENABLE_UTC=True,
```

## Example

[GoCelery GoDoc](https://godoc.org/github.com/gocelery/gocelery) has good examples.<br/>
Also take a look at `example` directory for sample python code.

### GoCelery Worker Example

Run Celery Worker implemented in Go

```go
// create redis connection pool
redisPool := &redis.Pool{
  Dial: func() (redis.Conn, error) {
		c, err := redis.DialURL("redis://")
		if err != nil {
			return nil, err
		}
		return c, err
	},
}

// initialize celery client
cli, _ := gocelery.NewCeleryClient(
	gocelery.NewRedisBroker(redisPool),
	&gocelery.RedisCeleryBackend{Pool: redisPool},
	5, // number of workers
)

// task
add := func(a, b int) int {
	return a + b
}

// register task
cli.Register("worker.add", add)

// start workers (non-blocking call)
cli.StartWorker()

// wait for client request
time.Sleep(10 * time.Second)

// stop workers gracefully (blocking call)
cli.StopWorker()
```

### Python Client Example

Submit Task from Python Client

```python
from celery import Celery

app = Celery('tasks',
    broker='redis://localhost:6379',
    backend='redis://localhost:6379'
)

@app.task
def add(x, y):
    return x + y

if __name__ == '__main__':
    ar = add.apply_async((5456, 2878), serializer='json')
    print(ar.get())
```

### Python Worker Example

Run Celery Worker implemented in Python

```python
from celery import Celery

app = Celery('tasks',
    broker='redis://localhost:6379',
    backend='redis://localhost:6379'
)

@app.task
def add(x, y):
    return x + y
```

```bash
celery -A worker worker --loglevel=debug --without-heartbeat --without-mingle
```

### GoCelery Client Example

Submit Task from Go Client

```go
// create redis connection pool
redisPool := &redis.Pool{
  Dial: func() (redis.Conn, error) {
		c, err := redis.DialURL("redis://")
		if err != nil {
			return nil, err
		}
		return c, err
	},
}

// initialize celery client
cli, _ := gocelery.NewCeleryClient(
	gocelery.NewRedisBroker(redisPool),
	&gocelery.RedisCeleryBackend{Pool: redisPool},
	1,
)

// prepare arguments
taskName := "worker.add"
argA := rand.Intn(10)
argB := rand.Intn(10)

// run task
asyncResult, err := cli.DelayV2(taskName, argA, argB)
// or delay with kwargs
// asyncResult, err := cli.DelayKwargsV2(taskName, kwargs map[string]interface{}, argA, argB)
if err != nil {
	panic(err)
}

// get results from backend with timeout
res, err := asyncResult.Get(10 * time.Second)
if err != nil {
	panic(err)
}

log.Printf("result: %+v of type %+v", res, reflect.TypeOf(res))
```

## Task Execution Modes

GoCelery supports two ways to register and execute tasks, each with different trade-offs:

### Mode 1: CeleryTask Interface (Recommended for Complex Tasks)

Implement the `CeleryTask` interface for zero-reflection execution and better type safety:

```go
type MyTask struct {
    A int
    B int
}

func (t *MyTask) ParseKwargs(kwargs map[string]interface{}) error {
    // Add nil check
    if kwargs == nil {
        return fmt.Errorf("kwargs cannot be nil")
    }

    // Extract and validate parameters
    aVal, aOk := kwargs["a"]
    bVal, bOk := kwargs["b"]
    if !aOk || !bOk {
        return fmt.Errorf("missing required keys: a, b")
    }

    // Type assertion with error handling
    aFloat, ok := aVal.(float64)
    if !ok {
        return fmt.Errorf("a must be a number")
    }
    bFloat, ok := bVal.(float64)
    if !ok {
        return fmt.Errorf("b must be a number")
    }

    t.A = int(aFloat)
    t.B = int(bFloat)
    return nil
}

func (t *MyTask) RunTask() (interface{}, error) {
    return t.A + t.B, nil
}

// Register the task
worker.Register("mytask", &MyTask{})
```

**Client submission (must use kwargs):**

```go
// Go Client
asyncResult, err := client.DelayKwargsV2("mytask", map[string]interface{}{
    "a": 10,
    "b": 20,
})

// Python Client
result = app.send_task(
    'mytask',
    kwargs={'a': 10, 'b': 20},
    serializer='json'
)
```

**Advantages:**

* ✅ Zero reflection overhead - direct method calls
* ✅ Compile-time type checking for task structure
* ✅ Support for complex parameter parsing and validation
* ✅ Better error handling capabilities
* ✅ Suitable for tasks with many parameters or complex logic

**Requirements:**

* Tasks **must** be submitted with `kwargs` (not positional args)
* Python clients must use `kwargs={'a': 10, 'b': 20}` format
* Go clients must use `DelayKwargsV2()` method

---

### Mode 2: Function Pointer (Simple & Convenient)

Register regular Go functions for straightforward tasks:

```go
add := func(a, b int) int {
    return a + b
}

worker.Register("worker.add", add)
```

**Client submission (positional args):**

```go
// Go Client
asyncResult, err := client.DelayV2("worker.add", 10, 20)

// Python Client
result = add.apply_async((10, 20), serializer='json')
```

**Advantages:**

* ✅ Simple and concise - no boilerplate code
* ✅ Natural Go function syntax
* ✅ Works with positional arguments
* ✅ Perfect for simple mathematical operations

**Limitations:**

* ❌ Runtime reflection overhead
* ❌ Parameters must match exactly (count and basic types)
* ❌ No support for kwargs - only positional args
* ❌ Limited type conversion (only `float64` to `int`/`float32`)
* ❌ Cannot handle complex parameter validation

---

### Comparison Table

| Aspect | CeleryTask Interface | Function Pointer |
| ------ | -------------------- | ---------------- |
| **Performance** | Fast (zero reflection) | Slower (reflection) |
| **Type Safety** | Compile-time | Runtime |
| **Parameter Style** | kwargs only | args only |
| **Validation** | Custom logic supported | Basic type coercion |
| **Code Complexity** | More boilerplate | Minimal code |
| **Best For** | Complex tasks, production services | Quick prototypes, simple math |
| **Client Method** | `DelayKwargsV2()` | `DelayV2()` / `Delay()` |

---

### Best Practices

1. **Use CeleryTask interface for production code** - Better performance and error handling
2. **Use function pointers for prototypes** - Quick testing and simple operations
3. **Never mix modes** - Don't try to use kwargs with function pointer tasks
4. **Always validate in ParseKwargs** - Check for nil and required keys
5. **Match client and worker styles** - Ensure client uses correct submission method

For protocol version differences, see [`.github/copilot-instructions.md`](.github/copilot-instructions.md).

## Sample Celery Task Message

Celery Message Protocol Version 2

```javascript
{
    "body": "W1s1NDU2LCAyODc4XSwge30sIHsiY2FsbGJhY2tzIjogbnVsbCwgImVycmJhY2tzIjogbnVsbCwgImNoYWluIjogbnVsbCwgImNob3JkIjogbnVsbH1d",
    "headers": {
        "lang": "py",
        "task": "worker.add",
        "id": "c8535050-68f1-4e18-9f32-f52f1aab6d9b",
        "root_id": "c8535050-68f1-4e18-9f32-f52f1aab6d9b",
        "parent_id": null,
        "group": null,
        "expires": null,
        "shadow": null,
        "retries": 0,
        "eta": null,
        "argsrepr": "[5456, 2878]",
        "timelimit": [60, null],
        "origin": "12345@hostname.local"
    },
    "properties": {
        "priority": 0,
        "body_encoding": "base64",
        "correlation_id": "c8535050-68f1-4e18-9f32-f52f1aab6d9b",
        "reply_to": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "delivery_info": {
            "routing_key": "celery",
            "exchange": ""
        },
        "delivery_mode": 2,
        "delivery_tag": "b2c3d4e5-f6a7-8901-bcde-f12345678901"
    },
    "content-type": "application/json",
    "content-encoding": "utf-8"
}
```

**Body decoded (base64):**

```javascript
[
    [5456, 2878],           // args
    {},                      // kwargs
    {                        // embed
        "callbacks": null,
        "errbacks": null,
        "chain": null,
        "chord": null
    }
]
```

## Projects

Please let us know if you use gocelery in your project!

## Contributing

You are more than welcome to make any contributions.
Please create Pull Request for any changes.

## LICENSE

The gocelery is offered under MIT license.
