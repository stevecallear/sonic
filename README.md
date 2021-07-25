# sonic
[![Build Status](https://github.com/stevecallear/sonic/actions/workflows/build.yml/badge.svg)](https://github.com/stevecallear/sonic/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/stevecallear/sonic/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/sonic)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/sonic)](https://goreportcard.com/report/github.com/stevecallear/sonic)

Sonic is a [Sonic](https://github.com/valeriansaliou/sonic) client for Go. It was created with the intention of offering a fully thread-safe client that can be shared across Go routines.

## Getting Started
```
go get github.com/stevecallear/sonic
```
```
search := sonic.NewSearch(sonic.Options{
    Addr:     "localhost:1491",
    Password: "password",
})
defer search.Close()

err := search.Ping()
if err != nil {
    log.Fatalln(err)
}
```

## Interface

### Close
All connections are terminated using the `Close` function as opposed to `Quit` seen in other clients. This is for consistency with the `io.Closer` interface.

### Flush
The `FLUSHC`, `FLUSHB` and `FLUSHO` commands are all handled using a single `Flush` function, with the appropriate command being identified from the supplied parameters. This is to simplify the interface and allow consistency with the behaviour of `Count`.

### Optional Parameters
Any parameter that is optional according to the [Sonic protocol](https://github.com/valeriansaliou/sonic/blob/master/PROTOCOL.md) can be omitted from the request struct. For example

```
search.Suggest(sonic.SuggestRequest{
    Collection: "collection",
    Bucket:     "bucket",
    Word:       "tex",
})
```
will result in `SUGGEST collection bucket "tex"` being sent over the wire, while

```
search.Suggest(sonic.SuggestRequest{
    Collection: "collection",
    Bucket:     "bucket",
    Word:       "tex",
    Limit:      5,
})
```
will result in `SUGGEST collection bucket "tex" LIMIT(5)` being sent.

## Connection Pool
By default created clients will share a single TCP connection. If the client is used by multiple Go routines then requests will block until the connection is available. If a connection is available within 30 seconds then `ErrPoolTimeout` will be returned.

The pool size can be configured to enable concurrent requests along with the timeout value.
```
ingest := sonic.NewIngest(sonic.Options{
    Addr:        "localhost:1491",
    Password:    "password",
    PoolSize:    4,
    PoolTimeout: 1 * time.Second,
})
```

## Examples

### Search
```
search := sonic.NewSearch(sonic.Options{
    Addr:     "localhost:1491",
    Password: "password",
})
defer search.Close()

res, err = search.Query(sonic.QueryRequest{
    Collection: "collection",
    Bucket:     "bucket",
    Terms:      "text",
})
if err != nil {
    log.Fatalln(err)
}

log.Println(res)
```

### Ingest
```
ingest := sonic.NewIngest(sonic.Options{
    Addr:     "localhost:1491",
    Password: "password",
})
defer ingest.Close()

err := ingest.Push(sonic.PushRequest{
    Collection: "collection",
    Bucket:     "bucket",
    Object:     "obj:id",
    Text:       "text",
})
if err != nil {
    log.Fatalln(err)
}
```

### Control
```
control := sonic.NewControl(sonic.Options{
    Addr:     "localhost:1491",
    Password: "password",
})
defer control.Close()

err := control.Trigger(sonic.TriggerRequest{
    Action: "consolidate",
})
if err != nil {
    log.Fatalln(err)
}
```

### Bulk
```
ingest := sonic.NewIngest(sonic.Options{
    Addr:        "localhost:1491",
    Password:    "password",
    PoolSize:    4,
    PoolTimeout: 1 * time.Second,
})
defer ingest.Close()

text := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

wg := new(sync.WaitGroup)
for _, t := range text {
    wg.Add(1)
    go func(text string) {
        defer wg.Done()

        err := ingest.Push(sonic.PushRequest{
            Collection: "collection",
            Bucket:     "bucket",
            Object:     "obj:id",
            Text:       text,
        })
        if err != nil {
            log.Fatalln(err)
        }
    }(t)
}

wg.Wait()
```