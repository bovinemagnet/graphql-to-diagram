# graphql-to-diagram
Convert a GraphQL Schema to a supported diagram type.


## Mermaid

```bash
go run main.go -format mermaid -schema schema.graphql -output output.mmd
```


The mermaid live editor can be used to view the diagram:

```bash
docker run --platform linux/amd64 --publish 8000:8080 ghcr.io/mermaid-js/mermaid-live-editor
```

Access the live editor at: http://localhost:8000/