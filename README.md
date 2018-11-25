# gocapture

Tool to find loop variables that are used in a go routine.
Examples:
```go
strings := []string{"a", "b", "c"}
for i := range strings {
    go func() {
        fmt.Println(strings[i])
    }
}
// but also more complex ones:
for i := range strings {
    if i > 1 {
        go func() {
            fmt.Println(strings[i])
        }
    }
}
```
The code is inspired by golang.org/x/tools/go/analysis/passes/loopclosure.
Uses the [golang tools analysis framework](golang.org/x/tools/go/analysis).

## Usage
Like any other go analysis tool: `$ gocapture <package>`.
(to check multiple packages: `folder/...`)