# curl-httpie-converter

GO CLI tool to convert between curl and HTTPie command formats.

## What it does

Converts HTTP requests between two popular formats:
- **curl** → **HTTPie**
- **HTTPie** → **curl**

Useful when you're working with different tools or sharing API examples with your team.

## Installation
```bash
go build -o converter main.go
```

## Usage
```bash
./converter -from curl -to httpie
```

Then paste your command and press `Ctrl+D` (or `Cmd+D` on Mac).

### Examples

**Convert curl to HTTPie:**
```bash
./converter -from curl -to httpie
```

Paste:
```
curl 'https://api.example.com/users' -X POST -H 'Content-Type: application/json' -H 'Authorization: Bearer token123' --data-raw '{"name":"John","age":30}'
```

Output:
```
http POST https://api.example.com/users Content-Type:application/json Authorization:'Bearer token123' name=John age:=30
```

**Convert HTTPie to curl:**
```bash
./converter -from httpie -to curl
```

Paste:
```
http POST https://api.example.com/users Authorization:'Bearer token123' name=John age:=30
```

Output:
```
curl https://api.example.com/users -X POST -H 'Authorization: Bearer token123' --data-raw '{"name":"John","age":30}'
```

## Flags

- `-from` - Input format (`curl` or `httpie`)
- `-to` - Output format (`curl` or `httpie`)

## Features

- Handles headers (Content-Type, Authorization, etc.)
- Converts JSON data between formats
- Supports nested JSON objects and arrays
- Preserves HTTP methods (GET, POST, PUT, DELETE, etc.)
- Cleans up whitespace and formatting

## Limitations

- Focuses on Content-Type and Authorization headers (other headers are ignored)
- Designed for JSON payloads
- Basic parsing - complex curl options may not be fully supported

## License
[Matltin](https://github.com/Matltin/)
